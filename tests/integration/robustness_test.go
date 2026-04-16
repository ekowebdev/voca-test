package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestRobustnessScenarios(t *testing.T) {
	TruncateTables()

	// Setup a primary user
	userBody := `{"name": "Robust User"}`
	wUser := PerformRequest("POST", "/api/v1/users", userBody)
	var userResp map[string]interface{}
	json.Unmarshal(wUser.Body.Bytes(), &userResp)
	userID := userResp["data"].(map[string]interface{})["id"].(string)

	// SCENARIO 1: Decimal Precision
	t.Run("Decimal Precision - Rounding & Minimum Unit", func(t *testing.T) {
		// Create wallet
		wBody := fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, userID)
		wReq := PerformRequest("POST", "/api/v1/wallets", wBody)
		var wResp map[string]interface{}
		json.Unmarshal(wReq.Body.Bytes(), &wResp)
		walletID := wResp["data"].(map[string]interface{})["id"].(string)

		// Topup 12.345 -> should be 12.35
		body := fmt.Sprintf(`{"amount": 12.345, "idempotency_key": "%s"}`, uuid.New().String())
		w := PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), body)
		if w.Code != http.StatusOK {
			t.Fatalf("Topup failed: %s", w.Body.String())
		}
		
		checkW := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s", walletID), "")
		var checkResp map[string]interface{}
		json.Unmarshal(checkW.Body.Bytes(), &checkResp)
		balStr := checkResp["data"].(map[string]interface{})["balance"].(string)
		assertDecimal(t, balStr, "12.35", "Precision rounding")

		// Payment 0.001 -> should be rejected
		payBody := fmt.Sprintf(`{"amount": 0.001, "idempotency_key": "%s"}`, uuid.New().String())
		wPay := PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/pay", walletID), payBody)
		if wPay.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for 0.001 payment, got %d", wPay.Code)
		}
	})

	// SCENARIO 2: Large Balances
	t.Run("Large Balances - Safely store 1B+", func(t *testing.T) {
		wBody := fmt.Sprintf(`{"user_id": "%s", "currency": "IDR"}`, userID)
		wReq := PerformRequest("POST", "/api/v1/wallets", wBody)
		var wResp map[string]interface{}
		json.Unmarshal(wReq.Body.Bytes(), &wResp)
		walletID := wResp["data"].(map[string]interface{})["id"].(string)

		amount := "1000000000.00" // 1B
		body := fmt.Sprintf(`{"amount": %s, "idempotency_key": "%s"}`, amount, uuid.New().String())
		PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), body)

		checkW := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s", walletID), "")
		var checkResp map[string]interface{}
		json.Unmarshal(checkW.Body.Bytes(), &checkResp)
		balStr := checkResp["data"].(map[string]interface{})["balance"].(string)
		assertDecimal(t, balStr, amount, "Large balance storage")
	})

	// SCENARIO 3: Currency Mismatch
	t.Run("Currency Mismatch - Reject transfers between USD and IDR", func(t *testing.T) {
        u2Body := `{"name": "Multi User"}`
        wU2 := PerformRequest("POST", "/api/v1/users", u2Body)
        var u2Resp map[string]interface{}
        json.Unmarshal(wU2.Body.Bytes(), &u2Resp)
        u2ID := u2Resp["data"].(map[string]interface{})["id"].(string)

        wUSD := PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, u2ID))
        var usdResp map[string]interface{}
        json.Unmarshal(wUSD.Body.Bytes(), &usdResp)
        usdID := usdResp["data"].(map[string]interface{})["id"].(string)

        wIDR := PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "IDR"}`, u2ID))
        var idrResp map[string]interface{}
        json.Unmarshal(wIDR.Body.Bytes(), &idrResp)
        idrID := idrResp["data"].(map[string]interface{})["id"].(string)

        // Topup USD
        PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", usdID), fmt.Sprintf(`{"amount": 100, "idempotency_key": "%s"}`, uuid.New().String()))

        // Transfer USD to IDR
        transferBody := fmt.Sprintf(`{"from_wallet_id": "%s", "to_wallet_id": "%s", "amount": 10, "idempotency_key": "%s"}`, usdID, idrID, uuid.New().String())
        wXfer := PerformRequest("POST", "/api/v1/wallets/transfer", transferBody)
        if wXfer.Code != http.StatusBadRequest {
            t.Errorf("Expected 400 for currency mismatch, got %d", wXfer.Code)
        }
    })

    // SCENARIO 4: Multiple Wallets Per User
    t.Run("Multiple Wallets Per User - One wallet per currency", func(t *testing.T) {
        u3Body := `{"name": "Max Wallet User"}`
        wU3 := PerformRequest("POST", "/api/v1/users", u3Body)
        var u3Resp map[string]interface{}
        json.Unmarshal(wU3.Body.Bytes(), &u3Resp)
        u3ID := u3Resp["data"].(map[string]interface{})["id"].(string)

        PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, u3ID))
        PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "IDR"}`, u3ID))
        
        // Create USD again (Fail)
        w3 := PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, u3ID))
        if w3.Code != http.StatusBadRequest {
            t.Errorf("Expected 400 for second USD wallet, got %d", w3.Code)
        }
    })

    // SCENARIO 5: Zero or Negative Amounts
    t.Run("Zero or Negative Amounts - Reject", func(t *testing.T) {
        TruncateTables()
        wU1 := PerformRequest("POST", "/api/v1/users", `{"name": "Test User"}`)
        var u1Resp map[string]interface{}
        json.Unmarshal(wU1.Body.Bytes(), &u1Resp)
        uID := u1Resp["data"].(map[string]interface{})["id"].(string)
        wReq := PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, uID))
        var wResp map[string]interface{}
        json.Unmarshal(wReq.Body.Bytes(), &wResp)
        walletID := wResp["data"].(map[string]interface{})["id"].(string)

        cases := []float64{0.00, -10.50}
        for _, amt := range cases {
            body := fmt.Sprintf(`{"amount": %f, "idempotency_key": "%s"}`, amt, uuid.New().String())
            res := PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), body)
            if res.Code != http.StatusBadRequest { t.Errorf("Expected 400 for amount %f in topup, got %d", amt, res.Code) }
            
            resPay := PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/pay", walletID), body)
            if resPay.Code != http.StatusBadRequest { t.Errorf("Expected 400 for amount %f in payment, got %d", amt, resPay.Code) }
        }
    })

    // SCENARIO 6: Duplicate Requests (Idempotency)
    t.Run("Duplicate Requests - Safely ignore double top-up", func(t *testing.T) {
        TruncateTables()
        wU := PerformRequest("POST", "/api/v1/users", `{"name": "Idem User"}`)
        var uResp map[string]interface{}
        json.Unmarshal(wU.Body.Bytes(), &uResp)
        uID := uResp["data"].(map[string]interface{})["id"].(string)
        wReq := PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, uID))
        var wResp map[string]interface{}
        json.Unmarshal(wReq.Body.Bytes(), &wResp)
        walletID := wResp["data"].(map[string]interface{})["id"].(string)

        idemKey := "UNIQUE-KEY-123"
        body := fmt.Sprintf(`{"amount": 100.00, "idempotency_key": "%s"}`, idemKey)
        PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), body)

        // Second request with same key
        res2 := PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), body)
        if res2.Code != http.StatusBadRequest { t.Errorf("Expected 400 for duplicate key, got %d", res2.Code) }

        checkW := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s", walletID), "")
        var checkResp map[string]interface{}
        json.Unmarshal(checkW.Body.Bytes(), &checkResp)
        assertDecimal(t, checkResp["data"].(map[string]interface{})["balance"].(string), "100.00", "Idempotency balance check")
    })

    // SCENARIO 7: Concurrent Spending
    t.Run("Concurrent Spending - Race condition prevention", func(t *testing.T) {
        TruncateTables()
        wU := PerformRequest("POST", "/api/v1/users", `{"name": "Race User"}`)
        var uResp map[string]interface{}
        json.Unmarshal(wU.Body.Bytes(), &uResp)
        uID := uResp["data"].(map[string]interface{})["id"].(string)
        wReq := PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, uID))
        var wResp map[string]interface{}
        json.Unmarshal(wReq.Body.Bytes(), &wResp)
        walletID := wResp["data"].(map[string]interface{})["id"].(string)
        
        PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), fmt.Sprintf(`{"amount": 100.00, "idempotency_key": "%s"}`, uuid.New().String()))

        // 6 simultaneous payments of 20.00 each on 100.00 balance
        var wg sync.WaitGroup
        numReqs := 6
        codes := make(chan int, numReqs)

        for i := 0; i < numReqs; i++ {
            wg.Add(1)
            go func(idx int) {
                defer wg.Done()
                body := fmt.Sprintf(`{"amount": 20.00, "idempotency_key": "CONC-%d"}`, idx)
                res := PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/pay", walletID), body)
                codes <- res.Code
            }(i)
        }
        wg.Wait()
        close(codes)

        success := 0
        for c := range codes {
            if c == http.StatusOK { success++ }
        }

        if success != 5 {
            t.Errorf("Expected 5 successes, got %d", success)
        }
    })

    // SCENARIO 8: Partial Failure During Transfer (Atomicity)
	t.Run("Partial Failure During Transfer - Atomicity", func(t *testing.T) {
		TruncateTables()
		wU1 := PerformRequest("POST", "/api/v1/users", `{"name": "U1"}`)
		var u1Resp map[string]interface{}
		json.Unmarshal(wU1.Body.Bytes(), &u1Resp)
		u1ID := u1Resp["data"].(map[string]interface{})["id"].(string)

		wU2 := PerformRequest("POST", "/api/v1/users", `{"name": "U2"}`)
		var u2Resp map[string]interface{}
		json.Unmarshal(wU2.Body.Bytes(), &u2Resp)
		u2ID := u2Resp["data"].(map[string]interface{})["id"].(string)

		w1 := PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, u1ID))
		var w1Resp map[string]interface{}
		json.Unmarshal(w1.Body.Bytes(), &w1Resp)
		wal1ID := w1Resp["data"].(map[string]interface{})["id"].(string)

		w2 := PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "IDR"}`, u2ID))
		var w2Resp map[string]interface{}
		json.Unmarshal(w2.Body.Bytes(), &w2Resp)
		wal2ID := w2Resp["data"].(map[string]interface{})["id"].(string)

		PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", wal1ID), fmt.Sprintf(`{"amount": 100.00, "idempotency_key": "%s"}`, uuid.New().String()))

		// Transfer with currency mismatch (will fail inside TX before commits)
		body := fmt.Sprintf(`{"from_wallet_id": "%s", "to_wallet_id": "%s", "amount": 50, "idempotency_key": "XFER-1"}`, wal1ID, wal2ID)
		res := PerformRequest("POST", "/api/v1/wallets/transfer", body)
		if res.Code == http.StatusOK {
			t.Fatalf("Expected failure for mismatch")
		}

		// Sanity check: Balance 1 must still be 100
		check := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s", wal1ID), "")
		var checkResp map[string]interface{}
		json.Unmarshal(check.Body.Bytes(), &checkResp)
		balStr := checkResp["data"].(map[string]interface{})["balance"].(string)
		assertDecimal(t, balStr, "100.00", "Atomic rollback check")
	})

    // SCENARIO 9: Ledger vs Balance Mismatch
    t.Run("Ledger vs Balance Mismatch - Reconciliation", func(t *testing.T) {
        TruncateTables()
        wU := PerformRequest("POST", "/api/v1/users", `{"name": "Recon User"}`)
        var uResp map[string]interface{}
        json.Unmarshal(wU.Body.Bytes(), &uResp)
        uID := uResp["data"].(map[string]interface{})["id"].(string)
        wReq := PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, uID))
        var wResp map[string]interface{}
        json.Unmarshal(wReq.Body.Bytes(), &wResp)
        walletID := wResp["data"].(map[string]interface{})["id"].(string)

        // Random ops
        PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), `{"amount": 100, "idempotency_key": "K1"}`)
        PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/pay", walletID), `{"amount": 30, "idempotency_key": "K2"}`)
        PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), `{"amount": 20, "idempotency_key": "K3"}`)

        // Recon
        checkW := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s", walletID), "")
        var checkResp map[string]interface{}
        json.Unmarshal(checkW.Body.Bytes(), &checkResp)
        balStr := checkResp["data"].(map[string]interface{})["balance"].(string)
        walletBal, _ := decimal.NewFromString(balStr)

        resLedger := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s/transactions", walletID), "")
        var ledgerResp map[string]interface{}
        json.Unmarshal(resLedger.Body.Bytes(), &ledgerResp)
        entries := ledgerResp["data"].([]interface{})
        sum := decimal.Zero
        for _, entry := range entries {
            e := entry.(map[string]interface{})
            amt, _ := decimal.NewFromString(e["amount"].(string))
            sum = sum.Add(amt)
        }

        if !sum.Equal(walletBal) {
            t.Errorf("Reconciliation failed: Ledger sum %s != Balance %s", sum, walletBal)
        }
    })

    // SCENARIO 10: Suspended Wallet Operations
    t.Run("Suspended Wallet Operations - Blocked", func(t *testing.T) {
        TruncateTables()
        wU := PerformRequest("POST", "/api/v1/users", `{"name": "Susp User"}`)
        var uResp map[string]interface{}
        json.Unmarshal(wU.Body.Bytes(), &uResp)
        uID := uResp["data"].(map[string]interface{})["id"].(string)
        wReq := PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, uID))
        var wResp map[string]interface{}
        json.Unmarshal(wReq.Body.Bytes(), &wResp)
        walletID := wResp["data"].(map[string]interface{})["id"].(string)

        PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/suspend", walletID), "")

        res := PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), `{"amount": 100, "idempotency_key": "K1"}`)
        if res.Code != http.StatusBadRequest { t.Errorf("Allowed topup on suspended wallet") }
    })

    // SCENARIO 11, 12: Consistency & Out-of-Order
    t.Run("Read-After-Write & Consistency", func(t *testing.T) {
        TruncateTables()
        wU := PerformRequest("POST", "/api/v1/users", `{"name": "Consist User"}`)
        var uResp map[string]interface{}
        json.Unmarshal(wU.Body.Bytes(), &uResp)
        uID := uResp["data"].(map[string]interface{})["id"].(string)
        wReq := PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, uID))
        var wResp map[string]interface{}
        json.Unmarshal(wReq.Body.Bytes(), &wResp)
        walletID := wResp["data"].(map[string]interface{})["id"].(string)

        for i := 1; i <= 5; i++ {
            body := fmt.Sprintf(`{"amount": 10, "idempotency_key": "OP-%d"}`, i)
            res := PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), body)
            if res.Code != http.StatusOK { t.Fatalf("Op %d failed", i) }
            
            // Read after write
            check := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s", walletID), "")
            var checkResp map[string]interface{}
            json.Unmarshal(check.Body.Bytes(), &checkResp)
            expected := fmt.Sprintf("%d.00", i*10)
            assertDecimal(t, checkResp["data"].(map[string]interface{})["balance"].(string), expected, fmt.Sprintf("Read-after-write op %d", i))
        }
    })

    // SCENARIO 13: Atomic Recovery Simulation
    t.Run("Atomic Recovery Simulation - Rollback validation", func(t *testing.T) {
        TruncateTables()
        wU := PerformRequest("POST", "/api/v1/users", `{"name": "Recovery User"}`)
        var uResp map[string]interface{}
        json.Unmarshal(wU.Body.Bytes(), &uResp)
        uID := uResp["data"].(map[string]interface{})["id"].(string)
        wReq := PerformRequest("POST", "/api/v1/wallets", fmt.Sprintf(`{"user_id": "%s", "currency": "USD"}`, uID))
        var wResp map[string]interface{}
        json.Unmarshal(wReq.Body.Bytes(), &wResp)
        walletID := wResp["data"].(map[string]interface{})["id"].(string)

        PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/topup", walletID), `{"amount": 100, "idempotency_key": "K1"}`)

        // Trigger a logic error (e.g., negative payment)
        res := PerformRequest("POST", fmt.Sprintf("/api/v1/wallets/%s/pay", walletID), `{"amount": -50, "idempotency_key": "K2"}`)
        if res.Code != http.StatusBadRequest { t.Errorf("Expected failure") }

        // Ensure balance hasn't changed
        check := PerformRequest("GET", fmt.Sprintf("/api/v1/wallets/%s", walletID), "")
        var checkResp map[string]interface{}
        json.Unmarshal(check.Body.Bytes(), &checkResp)
        assertDecimal(t, checkResp["data"].(map[string]interface{})["balance"].(string), "100.00", "Atomic rollback state check")
    })
}
