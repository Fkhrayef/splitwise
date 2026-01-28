# Splitwise Go API

> **Module:** `github.com/fkhayef/splitwise`

A Splitwise-like expense splitting API built with Go, demonstrating:
- **Dependency Injection** (Constructor Injection)
- **Factory Pattern** (Split Strategy Factory)
- **Strategy Pattern** (Even, Percentage, Exact splits)
- **Vertical Slicing** (Feature-based architecture)

## Project Structure

```
splitwise/
├── cmd/api/              # Application entry point
├── internal/
│   ├── config/           # Configuration management
│   ├── database/         # Database connection
│   ├── user/             # User feature (model, dto, repo, service, handler)
│   ├── group/            # Group feature
│   ├── expense/          # Expense feature
│   │   └── split/        # Split strategies (Strategy + Factory patterns)
│   ├── settlement/       # Settlement feature
│   └── notification/     # Notification feature
├── pkg/
│   ├── middleware/       # HTTP middlewares
│   └── response/         # Standard API responses
└── migrations/           # SQL migrations
```

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 14+

### Setup

1. **Clone and install dependencies:**
   ```bash
   go mod download
   ```

2. **Set up PostgreSQL:**
   ```bash
   # Create database
   createdb splitwise
   
   # Run migrations
   psql -d splitwise -f migrations/000001_init_schema.up.sql
   ```

3. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your database credentials
   ```

4. **Run the server:**
   ```bash
   go run cmd/api/main.go
   ```

## API Endpoints

### Users
- `POST   /api/v1/users` - Create user
- `GET    /api/v1/users` - List users
- `GET    /api/v1/users/{id}` - Get user
- `PUT    /api/v1/users/{id}` - Update user
- `DELETE /api/v1/users/{id}` - Delete user

### Groups
- `POST   /api/v1/groups` - Create group
- `GET    /api/v1/groups` - List my groups
- `GET    /api/v1/groups/{id}` - Get group with members
- `PUT    /api/v1/groups/{id}` - Update group
- `DELETE /api/v1/groups/{id}` - Delete group
- `POST   /api/v1/groups/{id}/members` - Add member
- `DELETE /api/v1/groups/{id}/members/{userId}` - Remove member
- `POST   /api/v1/groups/{id}/accept` - Accept invitation

### Expenses
- `POST   /api/v1/expenses` - Create expense
- `GET    /api/v1/expenses/{id}` - Get expense with splits
- `GET    /api/v1/expenses/group/{groupId}` - List group expenses
- `DELETE /api/v1/expenses/{id}` - Delete expense

### Split Operations
- `POST   /api/v1/expenses/splits/{splitId}/pay` - Mark split as paid
- `POST   /api/v1/expenses/splits/{splitId}/confirm` - Confirm payment
- `POST   /api/v1/expenses/splits/{splitId}/dispute` - Dispute split

### Settlements
- `POST   /api/v1/settlements` - Create settlement
- `GET    /api/v1/settlements` - List my settlements
- `GET    /api/v1/settlements/{id}` - Get settlement
- `POST   /api/v1/settlements/{id}/pay` - Mark as paid
- `POST   /api/v1/settlements/{id}/confirm` - Confirm receipt
- `POST   /api/v1/settlements/{id}/reject` - Reject settlement
- `GET    /api/v1/settlements/balances` - Get net balances

### Notifications
- `GET    /api/v1/notifications` - List notifications
- `GET    /api/v1/notifications/unread-count` - Get unread count
- `POST   /api/v1/notifications/{id}/read` - Mark as read
- `POST   /api/v1/notifications/read-all` - Mark all as read

## Split Types

### EVEN Split
Divides expense equally among all participants.

```json
{
  "group_id": 1,
  "description": "Dinner",
  "amount": 100.00,
  "split_type": "EVEN",
  "participants": [
    {"user_id": 1},
    {"user_id": 2},
    {"user_id": 3}
  ]
}
```

### PERCENTAGE Split
Divides based on specified percentages (must sum to 100).

```json
{
  "group_id": 1,
  "description": "Rent",
  "amount": 1000.00,
  "split_type": "PERCENTAGE",
  "participants": [
    {"user_id": 1, "percentage": 50},
    {"user_id": 2, "percentage": 30},
    {"user_id": 3, "percentage": 20}
  ]
}
```

### EXACT Split
Each participant owes a specific amount (must sum to total).

```json
{
  "group_id": 1,
  "description": "Shopping",
  "amount": 150.00,
  "split_type": "EXACT",
  "participants": [
    {"user_id": 1, "amount": 50.00},
    {"user_id": 2, "amount": 75.00},
    {"user_id": 3, "amount": 25.00}
  ]
}
```

## Design Patterns

### Dependency Injection
All dependencies are injected via constructors:

```go
// Repository depends on database
userRepo := user.NewRepository(db)

// Service depends on repository
userService := user.NewService(userRepo)

// Handler depends on service
userHandler := user.NewHandler(userService)
```

### Strategy Pattern
Different split calculation algorithms:

```go
type Strategy interface {
    Calculate(totalAmount float64, payerID int64, participants []SplitInput) ([]SplitOutput, error)
    Type() SplitType
    Validate(totalAmount float64, participants []SplitInput) error
}
```

### Factory Pattern
Creates appropriate strategy based on type:

```go
strategy, err := splitFactory.Create(SplitTypeEven)
// or
strategy, err := splitFactory.CreateFromString("PERCENTAGE")
```

## License

MIT
