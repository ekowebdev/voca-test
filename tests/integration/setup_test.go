package integration

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"voca-test/internal/db"
	"voca-test/internal/handler"
	"voca-test/internal/repository"
	"voca-test/internal/routes"
	"voca-test/internal/service"
	"voca-test/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var testRouter *gin.Engine
var testPool *pgxpool.Pool

func init() {
	// 1. Load config
	cfg := util.LoadConfig("../../.env.testing")

	// 2. Connect to DB
	database, err := db.ConnectPostgres(cfg)
	if err != nil {
		log.Fatalf("Integration Test: Could not connect to database: %v", err)
	}
	testPool = database.Pool

	// 3. Initialize Repositories
	userRepo := repository.NewUserRepository(testPool)
	walletRepo := repository.NewWalletRepository(testPool)
	ledgerRepo := repository.NewLedgerRepository(testPool)
	idempotencyRepo := repository.NewIdempotencyRepository(testPool)

	// 4. Initialize Services
	userService := service.NewUserService(userRepo)
	walletService := service.NewWalletService(
		testPool,
		userRepo,
		walletRepo,
		ledgerRepo,
		idempotencyRepo,
	)
	ledgerService := service.NewLedgerService(walletRepo, ledgerRepo)

	// 5. Initialize Handlers
	userHandler := handler.NewUserHandler(userService)
	walletHandler := handler.NewWalletHandler(walletService)
	ledgerHandler := handler.NewLedgerHandler(ledgerService)

	// 6. Setup Router & Routes
	gin.SetMode(gin.TestMode)
	testRouter = gin.New()
	routes.SetupRoutes(testRouter, cfg, userHandler, walletHandler, ledgerHandler)
}

// TruncateTables cleans up the database
func TruncateTables() {
	tables := []string{"idempotency_keys", "ledger", "wallets", "users"}
	query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", strings.Join(tables, ", "))
	_, err := testPool.Exec(context.Background(), query)
	if err != nil {
		log.Fatalf("Failed to truncate tables: %v", err)
	}
}

// PerformRequest as a helper to call the router
func PerformRequest(method, path string, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

// assertDecimal helper to compare decimals safely in tests
func assertDecimal(t *testing.T, gotStr, wantStr, context string) {
	t.Helper()
	got, err1 := decimal.NewFromString(gotStr)
	want, err2 := decimal.NewFromString(wantStr)
	if err1 != nil || err2 != nil {
		t.Errorf("%s: Error parsing decimals: got=%v, want=%v", context, err1, err2)
		return
	}
	if !got.Equal(want) {
		t.Errorf("%s: Expected balance %s, got %s", context, want, got)
	}
}
