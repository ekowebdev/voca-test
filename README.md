# Multi-Currency E-Wallet Backend

This is a simplified multi-currency E-Wallet backend system implemented in Go using the Repository pattern, Gin framework, and PostgreSQL. It features a ledger-based audit trail, safe decimal handling for financial operations, and robust concurrency control.

## Tech Stack
- **Language**: Go 1.25.0
- **Database**: PostgreSQL 14
- **Framework**: Gin (HTTP Routing & Request handling)
- **Decimal library**: `github.com/shopspring/decimal`
- **UUID generator**: `github.com/google/uuid`
- **DB Driver/Pool**: `github.com/jackc/pgx/v5`
- **API Documentation**: `github.com/swaggo/swag` (Swagger)
- **Testing**: `github.com/stretchr/testify` (Assertions and Mocks)

## Project Structure
- `cmd/api/main.go`: Application entry point and dependency wiring.
- `internal/models`: Core business entities (User, Wallet, Ledger).
- `internal/repository`: Interface-based persistence layer using PostgreSQL.
- `internal/service`: Core business logic, transaction handling, and business rules.
- `internal/handler`: HTTP request/response handlers using Gin.
- `internal/middleware`: Modular request processing logic (Logging, Recovery, CORS, Security).
- `migrations/schema.sql`: Database schema definition with triggers for audit.
- `docs`: Auto-generated Swagger API documentation files.

## Key Features & Best Practices
1. **Ledger-Based Audit**: Every single wallet balance change (Top-up, Payment, Transfer) is recorded in an append-only `ledger` table before the balance is updated.
2. **Safe Decimal Arithmetic**: All financial amounts are handled using the `decimal.Decimal` library to avoid floating-point precision errors (e.g., `0.1 + 0.2 != 0.3`).
3. **Concurrency Control**: 
   - Uses **Pessimistic Locking** (`SELECT ... FOR UPDATE`) at the row level. When a wallet balance is being updated, other concurrent requests for the *same* wallet must wait, preventing race conditions.
   - For **Transfers**, a deterministic locking order (locking smaller UUID first) is used to prevent deadlocks.
4. **Idempotency**: All write operations (Top-up, Payment, Transfer) require an `idempotency_key` to safely handle duplicate requests from the client.
5. **Atomicity**: Multi-wallet operations (like transfers) are executed within a single SQL transaction. If any part of the operation fails, all changes are rolled back.
6. **Rounding Logic**: All amounts are automatically rounded to 2 decimal places (e.g., `12.3456` becomes `12.35`).

## How to Run

### 1. Prerequisites
- **Go 1.25.0** or later.
- **PostgreSQL 14** or later.
- **Air** (optional, for live reload).

### 2. Environment Setup
1.  Copy the example environment file:
    ```bash
    cp .env.example .env
    ```
2.  Edit `.env` and update the database configuration to match your local PostgreSQL setup.

### 3. Database Setup
You need to create the database and apply the initial schema:

1.  **Create the Database**:
    ```sql
    CREATE DATABASE wallet;
    ```
2.  **Apply Schema**:
    Import the schema located at `migrations/schema.sql`:
    ```bash
    psql -U postgres -d wallet -f migrations/schema.sql
    ```

### 4. Install Dependencies
```bash
go mod download
```

### 5. Run the Application
You can run the application directly:
```bash
go run cmd/api/main.go
```
The server will start on `http://localhost:8080`.

### 6. Development with Live Reload
For development, it is recommended to use **Air**:
1.  **Install Air** (if not already installed):
    ```bash
    go install github.com/air-verse/air@latest
    ```
2.  **Run with Air**:
    ```bash
    air
    ```

## API Documentation (Swagger)

The project includes an interactive Swagger UI to explore and test the API.

### 1. Accessing Swagger UI
1. Run the application (using `go run` or `air`).
2. Open your browser and navigate to: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

### 2. Regenerating Documentation
If you make changes to the API handlers or models, you need to regenerate the Swagger files:
1. Ensure the `swag` CLI is installed:
   ```bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ```
2. Run the generation command:
   ```bash
   swag init -g cmd/api/main.go --pd
   ```

## Testing

The project includes unit and integration tests. Integration tests require a dedicated test database to ensure data isolation.

### 1. Test Database Setup
1.  **Create the Test Database**:
    ```sql
    CREATE DATABASE wallet_testing;
    ```
2.  **Apply Schema to Test DB**:
    ```bash
    psql -U postgres -d wallet_testing -f migrations/schema.sql
    ```
3.  **Configure Test Environment**:
    Ensure `.env.testing` has the correct credentials for `wallet_testing`.

### 2. Running Tests

#### Unit Tests
Unit tests are fast and do not require a database connection. They focus on business logic and utility functions.
```bash
# Run unit tests only (excluding integration folder)
go test ./internal/... -v
```

#### Integration Tests
Integration tests require a running database (see [Test Database Setup](#1-test-database-setup) above).
```bash
# Run integration tests specifically
go test ./tests/integration/... -v
```

#### All Tests
To run everything (requires database for integration tests):
```bash
go test ./... -v
```

## Standard Response Format

All API responses follow a consistent structure to ensure ease of integration.

### Success Response
```json
{
  "status": "success",
  "message": "Operation successful description",
  "data": { ... } // Optional: contains the requested data
}
```

### Error Response
```json
{
  "status": "error",
  "message": "Invalid request body",
  "errors": {
    "user_id": "must be a valid UUID",
    "currency": "is required and cannot be empty"
  }
}
```

## API Documentation

### Users
- `POST /api/v1/users`: Create a new user.
  - Body: `{"name": "John Doe"}`

### Wallets
- `POST /api/v1/wallets`: Create a wallet for a user.
  - Body: `{"user_id": "<uuid>", "currency": "USD"}`
- `GET /api/v1/wallets/:id`: Get wallet balance and status.
- `GET /api/v1/wallets/:id/transactions`: Get wallet transaction history.
- `POST /api/v1/wallets/:id/topup`: Top-up money.
  - Body: `{"amount": 1000.50, "idempotency_key": "unique-uuid-1"}`
- `POST /api/v1/wallets/:id/pay`: Spend money.
  - Body: `{"amount": 200.10, "idempotency_key": "unique-uuid-2"}`
- `POST /api/v1/wallets/:id/suspend`: Suspend a wallet.
- `POST /api/v1/wallets/transfer`: Move money between same-currency wallets.
  - Body: `{"from_wallet_id": "<uuid>", "to_wallet_id": "<uuid>", "amount": 300.40, "idempotency_key": "unique-uuid-3"}`
