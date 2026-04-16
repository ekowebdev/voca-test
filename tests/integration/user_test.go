package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateUser(t *testing.T) {
	TruncateTables()

	t.Run("Successfully Create User", func(t *testing.T) {
		body := `{"name": "John Doe"}`
		w := PerformRequest("POST", "/api/v1/users", body)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp["message"] != "User created successfully" {
			t.Errorf("Expected success message, got %v", resp["message"])
		}

		data := resp["data"].(map[string]interface{})
		if data["name"] != "John Doe" {
			t.Errorf("Expected name John Doe, got %v", data["name"])
		}
	})

	t.Run("Fail Create User - Empty Name", func(t *testing.T) {
		body := `{"name": ""}`
		w := PerformRequest("POST", "/api/v1/users", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}
