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
		meta := resp["meta"].(map[string]interface{})
		summary := resp["summary"].(map[string]interface{})

		if len(data) != 2 {
			t.Errorf("Expected 2 transactions, got %d", len(data))
		}

		if meta["total_items"].(float64) != 2 {
			t.Errorf("Expected total_items 2, got %v", meta["total_items"])
		}

		if summary["total_credit"] == nil || summary["total_debit"] == nil {
			t.Errorf("Expected summary to have total_credit and total_debit")
		}

		// Verify links
		links := meta["links"].(map[string]interface{})
		if links["current"] == "" || links["first"] == "" || links["last"] == "" {
			t.Errorf("Expected pagination links to be populated, got %v", links)
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

	t.Run("Get Transaction History with Type Filter and Summary", func(t *testing.T) {
		// Filter by TOPUP
		w := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s/transactions?type=TOPUP", walletID), "")

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		summary := resp["summary"].(map[string]interface{})

		if len(data) != 1 {
			t.Errorf("Expected 1 transaction (TOPUP), got %d", len(data))
		}

		if data[0].(map[string]interface{})["type"] != "TOPUP" {
			t.Errorf("Expected transaction to be TOPUP, got %v", data[0].(map[string]interface{})["type"])
		}

		// Verify summary for TOPUP only
		// Convert to string first as decimal might be numerics or string in map
		creditStr := fmt.Sprintf("%v", summary["total_credit"])
		debitStr := fmt.Sprintf("%v", summary["total_debit"])
		
		gotCredit, _ := decimal.NewFromString(creditStr)
		gotDebit, _ := decimal.NewFromString(debitStr)
		
		if !gotCredit.Equal(decimal.NewFromInt(50000)) {
			t.Errorf("Expected total_credit 50000, got %s", gotCredit)
		}
		if !gotDebit.IsZero() {
			t.Errorf("Expected total_debit 0 for TOPUP filter, got %s", gotDebit)
		}
	})

	t.Run("Get Transaction History with Pagination", func(t *testing.T) {
		// page=1, per_page=1 -> should return 1 item (PAYMENT)
		w := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s/transactions?page=1&per_page=1", walletID), "")

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		data := resp["data"].([]interface{})
		meta := resp["meta"].(map[string]interface{})

		if len(data) != 1 {
			t.Errorf("Expected 1 transaction, got %d", len(data))
		}

		if meta["total_items"].(float64) != 2 {
			t.Errorf("Expected total_items 2, got %v", meta["total_items"])
		}

		// Verify links for paginated request
		links := meta["links"].(map[string]interface{})
		if links["next"] == "" {
			t.Errorf("Expected next link to be populated for page 1 of 2")
		}
		if links["prev"] != nil {
			t.Logf("Prev link: %v", links["prev"])
		}
		if meta["current_page"].(float64) != 1 {
			t.Errorf("Expected current_page 1, got %v", meta["current_page"])
		}
		if meta["total_pages"].(float64) != 2 {
			t.Errorf("Expected total_pages 2, got %v", meta["total_pages"])
		}

		// page=2, per_page=1 -> should return 1 item (TOPUP)
		w2 := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s/transactions?page=2&per_page=1", walletID), "")
		if w2.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w2.Code)
		}
		json.Unmarshal(w2.Body.Bytes(), &resp)
		data2 := resp["data"].([]interface{})

		if len(data2) != 1 {
			t.Errorf("Expected 1 transaction for page 2, got %d", len(data2))
		}
		if data2[0].(map[string]interface{})["type"] != "TOPUP" {
			t.Errorf("Expected second page to have TOPUP, got %v", data2[0].(map[string]interface{})["type"])
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
