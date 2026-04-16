package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestLedgerHistory(t *testing.T) {
	TruncateTables()

	// 1. Setup User & Wallet
	userBody := `{"name": "Ledger User"}`
	wUser := PerformRequest("POST", "/api/v1/users", userBody)
	var userResp map[string]interface{}
	json.Unmarshal(wUser.Body.Bytes(), &userResp)
	userID := userResp["data"].(map[string]interface{})["id"].(string)

	wBody := fmt.Sprintf(`{"user_id": "%s", "currency": "IDR"}`, userID)
	wReq := PerformRequest("POST", "/api/v1/wallets", wBody)
	var wResp map[string]interface{}
	json.Unmarshal(wReq.Body.Bytes(), &wResp)
	walletID := wResp["data"].(map[string]interface{})["id"].(string)

	// 2. Perform some operations
	// TopUp 50000
	PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), 
		fmt.Sprintf(`{"amount": 50000, "idempotency_key": "%s"}`, uuid.New().String()))
	
	// Pay 15000
	PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/pay", walletID), 
		fmt.Sprintf(`{"amount": 15000, "idempotency_key": "%s"}`, uuid.New().String()))

	t.Run("Get Transaction History Success", func(t *testing.T) {
		w := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s/transactions", walletID), "")

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})

		if len(data) != 2 {
			t.Errorf("Expected 2 transactions, got %d", len(data))
		}

		// Most recent should be first (Payment)
		first := data[0].(map[string]interface{})
		if first["type"] != "PAYMENT" {
			t.Errorf("Expected first transaction to be PAYMENT, got %v", first["type"])
		}

		// decimal.Decimal usually marshals to string
		amountStr := first["amount"].(string)
		gotAmount, _ := decimal.NewFromString(amountStr)
		wantAmount := decimal.NewFromFloat(-15000)
		if !gotAmount.Equal(wantAmount) {
			t.Errorf("Expected first amount %s, got %s", wantAmount, gotAmount)
		}
	})

	t.Run("Get History - Non-existent Wallet", func(t *testing.T) {
		fakeID := uuid.New().String()
		w := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s/transactions", fakeID), "")

		if w.Code != http.StatusInternalServerError { // Based on current service implementation returning error
			t.Errorf("Expected error status for fake wallet, got %d", w.Code)
		}
	})
}
