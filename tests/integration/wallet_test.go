package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestWalletOperations(t *testing.T) {
	TruncateTables()

	// 1. Setup User
	userBody := `{"name": "Wallet User"}`
	wUser := PerformRequest("POST", "/api/v1/users", userBody)
	var userResp map[string]interface{}
	json.Unmarshal(wUser.Body.Bytes(), &userResp)
	userData := userResp["data"].(map[string]interface{})
	userID := userData["id"].(string)

	var walletID string

	t.Run("Create Wallet Successfully", func(t *testing.T) {
		body := fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, userID)
		w := PerformRequest("POST", "/api/v1/wallets", body)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		walletID = data["id"].(string)

		if data["currency"] != "USD" {
			t.Errorf("Expected currency USD, got %v", data["currency"])
		}
	})

	t.Run("Fail Create Duplicate Wallet", func(t *testing.T) {
		body := fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, userID)
		w := PerformRequest("POST", "/api/v1/wallets", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for duplicate wallet, got %d", w.Code)
		}
	})

	t.Run("TopUp Successfully", func(t *testing.T) {
		idemKey := uuid.New().String()
		body := fmt.Sprintf(`{"amount": 100.50, "idempotency_key": "%s"}`, idemKey)
		w := PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})

		gotBal, _ := decimal.NewFromString(data["balance"].(string))
		wantBal := decimal.NewFromFloat(100.50)
		if !gotBal.Equal(wantBal) {
			t.Errorf("Expected balance %s, got %s", wantBal, gotBal)
		}

		// Test Idempotency
		wDup := PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), body)
		if wDup.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for duplicate idempotency key, got %d", wDup.Code)
		}
	})

	t.Run("Payment Successfully", func(t *testing.T) {
		idemKey := uuid.New().String()
		body := fmt.Sprintf(`{"amount": 50.25, "idempotency_key": "%s"}`, idemKey)
		w := PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/pay", walletID), body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})

		// 100.50 - 50.25 = 50.25
		gotBal, _ := decimal.NewFromString(data["balance"].(string))
		wantBal := decimal.NewFromFloat(50.25)
		if !gotBal.Equal(wantBal) {
			t.Errorf("Expected balance %s, got %s", wantBal, gotBal)
		}
	})

	t.Run("Fail Payment - Insufficient Funds", func(t *testing.T) {
		idemKey := uuid.New().String()
		body := fmt.Sprintf(`{"amount": 1000.00, "idempotency_key": "%s"}`, idemKey)
		w := PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/pay", walletID), body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("Transfer Successfully", func(t *testing.T) {
		// Setup Receiver
		user2Body := `{"name": "Receiver User"}`
		wUser2 := PerformRequest("POST", "/api/v1/users", user2Body)
		var user2Resp map[string]interface{}
		json.Unmarshal(wUser2.Body.Bytes(), &user2Resp)
		user2Data := user2Resp["data"].(map[string]interface{})
		user2ID := user2Data["id"].(string)

		w2Body := fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, user2ID)
		w2Req := PerformRequest("POST", "/api/v1/wallets", w2Body)
		var w2Resp map[string]interface{}
		json.Unmarshal(w2Req.Body.Bytes(), &w2Resp)
		w2Data := w2Resp["data"].(map[string]interface{})
		wallet2ID := w2Data["id"].(string)

		// Perform Transfer
		idemKey := uuid.New().String()
		transferBody := fmt.Sprintf(`{
			"from_wallet_id": "%s",
			"to_wallet_id": "%s",
			"amount": 20.00,
			"idempotency_key": "%s"
		}`, walletID, wallet2ID, idemKey)

		w := PerformRequest("POST", "/api/v1/wallets/transfer", transferBody)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Verify Balances
		w1Check := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s", walletID), "")
		var w1Data map[string]interface{}
		json.Unmarshal(w1Check.Body.Bytes(), &w1Data)
		gotBal1, _ := decimal.NewFromString(w1Data["data"].(map[string]interface{})["balance"].(string))
		wantBal1 := decimal.NewFromFloat(30.25)
		if !gotBal1.Equal(wantBal1) {
			t.Errorf("Expected sender balance %s, got %s", wantBal1, gotBal1)
		}

		w2Check := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s", wallet2ID), "")
		var w2DataFinal map[string]interface{}
		json.Unmarshal(w2Check.Body.Bytes(), &w2DataFinal)
		gotBal2, _ := decimal.NewFromString(w2DataFinal["data"].(map[string]interface{})["balance"].(string))
		wantBal2 := decimal.NewFromFloat(20.00)
		if !gotBal2.Equal(wantBal2) {
			t.Errorf("Expected receiver balance %s, got %s", wantBal2, gotBal2)
		}
	})
}
