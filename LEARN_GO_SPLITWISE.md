# 📚 Complete Guide: Building a Splitwise API with Go

> **A comprehensive, educational guide to understanding every detail of this project.**
>
> This document teaches you Go concepts, design patterns, and best practices through the lens of a real-world expense-splitting application.

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Prerequisites & Setup](#2-prerequisites--setup)
3. [Go Fundamentals for This Project](#3-go-fundamentals-for-this-project)
4. [Project Architecture](#4-project-architecture)
5. [Design Patterns Deep Dive](#5-design-patterns-deep-dive)
6. [Layer-by-Layer Breakdown](#6-layer-by-layer-breakdown)
7. [Database Schema](#7-database-schema)
8. [API Endpoints Reference](#8-api-endpoints-reference)
9. [Split Strategies (Strategy Pattern)](#9-split-strategies-strategy-pattern)
10. [Settlement System](#10-settlement-system)
11. [User Identification](#11-user-identification)
12. [How to Extend This Project](#12-how-to-extend-this-project)
13. [Common Go Patterns & Idioms](#13-common-go-patterns--idioms)
14. [Testing Your Code](#14-testing-your-code)
15. [Troubleshooting Guide](#15-troubleshooting-guide)
16. [Glossary](#16-glossary)

---

# 1. Introduction

## What We're Building

This is a **Splitwise-like API** - an application that helps groups of people split expenses fairly. When roommates share dinner, travel buddies split hotel costs, or friends go on trips together, this app tracks who paid what and who owes whom.

## What You'll Learn

- **Go Language Fundamentals** - structs, interfaces, packages, error handling
- **Dependency Injection** - how to wire components together without tight coupling
- **Factory Pattern** - creating objects without specifying exact types
- **Strategy Pattern** - swapping algorithms at runtime
- **Repository Pattern** - abstracting data access
- **Vertical Slicing** - organizing code by feature, not by layer
- **REST API Design** - building clean, consistent endpoints
- **PostgreSQL Integration** - working with databases in Go

## Why Go?

Go (Golang) is excellent for building APIs because:

1. **Fast compilation** - see changes instantly
2. **Strong typing** - catch errors at compile time
3. **Built-in concurrency** - handle many requests efficiently
4. **Simple syntax** - easy to read and maintain
5. **Great standard library** - `net/http` is production-ready
6. **Single binary deployment** - no runtime dependencies

---

# 2. Prerequisites & Setup

## Required Software

| Software     | Version | Purpose                      |
| ------------ | ------- | ---------------------------- |
| Go           | 1.21+   | The programming language     |
| PostgreSQL   | 14+     | Database (can run in Docker) |
| Git          | Any     | Version control              |
| Postman/curl | Any     | API testing                  |

## Project Setup Commands

```bash
# 1. Create project directory
mkdir splitwise && cd splitwise

# 2. Initialize Go module
go mod init github.com/yourusername/splitwise

# 3. Install dependencies
go get github.com/go-chi/chi/v5      # HTTP router
go get github.com/lib/pq              # PostgreSQL driver
go get github.com/joho/godotenv       # Environment variables

# 4. Verify dependencies
go mod tidy
```

## Environment Configuration

Create a `.env` file in your project root:

```env
# Database connection string
DATABASE_URL=postgres://postgres:postgres@localhost:5432/splitwise?sslmode=disable

# Server port
PORT=8080
```

## Database Setup

```bash
# Using Docker (recommended)
docker run --name pg-local -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:alpine

# Create the database
docker exec -it pg-local psql -U postgres -c "CREATE DATABASE splitwise;"

# Run migrations
docker exec -i pg-local psql -U postgres -d splitwise < migrations/000001_init_schema.up.sql
```

---

# 3. Go Fundamentals for This Project

## 3.1 Packages

In Go, code is organized into **packages**. Every Go file starts with a `package` declaration.

```go
package user  // This file belongs to the "user" package
```

**Key Rules:**

- All files in a directory must have the same package name
- Package name usually matches the directory name
- `package main` is special - it's the entry point of your program

**In Our Project:**

```
internal/user/       → package user
internal/group/      → package group
internal/expense/    → package expense
cmd/api/             → package main
```

## 3.2 Imports

```go
import (
    "fmt"                           // Standard library
    "net/http"                      // Standard library

    "github.com/go-chi/chi/v5"      // Third-party package

    "github.com/yourname/splitwise/internal/user"  // Your own package
)
```

**Import Aliases:**

```go
import (
    mw "github.com/yourname/splitwise/pkg/middleware"  // Alias: use mw.Function()
    _ "github.com/lib/pq"                               // Blank import: runs init() only
)
```

## 3.3 Structs

Structs are Go's way of creating custom types (like classes in other languages, but without inheritance).

```go
// Define a struct
type User struct {
    ID        int64     `json:"id"`          // Field with JSON tag
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

// Create an instance
user := User{
    ID:       1,
    Username: "john_doe",
    Email:    "john@example.com",
}

// Access fields
fmt.Println(user.Username)  // "john_doe"
```

**Struct Tags:**

```go
`json:"username"`           // JSON serialization name
`json:"avatar,omitempty"`   // Omit if empty/zero value
`validate:"required,email"` // Validation rules
```

## 3.4 Methods

Methods are functions attached to types:

```go
// Method on User struct
func (u *User) ToResponse() *UserResponse {
    return &UserResponse{
        ID:       u.ID,
        Username: u.Username,
        Email:    u.Email,
    }
}

// Usage
user := &User{ID: 1, Username: "john"}
response := user.ToResponse()
```

**Receiver Types:**

- `(u User)` - Value receiver (copies the struct)
- `(u *User)` - Pointer receiver (modifies original, more efficient for large structs)

**Rule of Thumb:** Use pointer receivers when:

1. The method modifies the struct
2. The struct is large
3. Consistency (if one method uses pointer, all should)

## 3.5 Interfaces

Interfaces define behavior (what methods a type must have):

```go
// Define an interface
type Strategy interface {
    Calculate(total float64, payerID int64, participants []SplitInput) ([]SplitOutput, error)
    Type() SplitType
}

// Any struct with these methods "implements" the interface
type EvenStrategy struct{}

func (s *EvenStrategy) Calculate(...) ([]SplitOutput, error) { ... }
func (s *EvenStrategy) Type() SplitType { return SplitTypeEven }

// EvenStrategy now implements Strategy interface - NO explicit declaration needed!
```

**Go's Interface Philosophy:**

- Interfaces are **implicitly implemented** (no `implements` keyword)
- If a type has all the methods, it automatically satisfies the interface
- This enables loose coupling and easy testing

## 3.6 Error Handling

Go doesn't have exceptions. Functions return errors explicitly:

```go
// Function that can fail
func (r *Repository) GetByID(ctx context.Context, id int64) (*User, error) {
    user := &User{}
    err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Username)

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, nil  // Not found (not an error)
        }
        return nil, fmt.Errorf("failed to get user: %w", err)  // Wrap error
    }

    return user, nil
}

// Calling the function
user, err := repo.GetByID(ctx, 123)
if err != nil {
    // Handle error
    log.Printf("Error: %v", err)
    return
}
if user == nil {
    // Handle not found
    return
}
// Use user...
```

**Error Wrapping:**

```go
// Wrap errors to add context
return nil, fmt.Errorf("failed to create user: %w", err)

// Check wrapped errors
if errors.Is(err, ErrUserNotFound) {
    // Handle specific error
}
```

## 3.7 Pointers

Pointers hold memory addresses:

```go
// Value
x := 5       // x is 5

// Pointer
p := &x      // p points to x's memory address
*p = 10      // Dereference: change value at that address
// Now x is 10

// Why use pointers?
func updateUser(u *User) {
    u.Name = "New Name"  // Modifies original
}

func copyUser(u User) {
    u.Name = "New Name"  // Modifies a copy (original unchanged)
}
```

**In Our Project:**

- `*sql.DB` - pointer to database connection (shared)
- `*User` - pointer to user (avoid copying)
- `*string` - pointer to string (allows nil for optional fields)

## 3.8 Context

Context carries deadlines, cancellation signals, and request-scoped values:

```go
func (r *Repository) GetByID(ctx context.Context, id int64) (*User, error) {
    // Pass context to database query - allows cancellation
    row := r.db.QueryRowContext(ctx, query, id)
    ...
}

// In HTTP handler, context comes from request
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
    user, err := h.service.GetByID(r.Context(), id)
    ...
}
```

---

# 4. Project Architecture

## 4.1 Vertical Slicing

Traditional layered architecture organizes by **technical concern**:

```
❌ Layered (Horizontal)
├── controllers/
│   ├── user_controller.go
│   ├── group_controller.go
│   └── expense_controller.go
├── services/
│   ├── user_service.go
│   └── ...
└── repositories/
    ├── user_repository.go
    └── ...
```

Our project uses **vertical slicing** - organized by **feature**:

```
✅ Vertical Slicing
├── internal/
│   ├── user/           # Everything about users
│   │   ├── model.go
│   │   ├── dto.go
│   │   ├── repository.go
│   │   ├── service.go
│   │   └── handler.go
│   ├── group/          # Everything about groups
│   │   └── ...
│   ├── expense/        # Everything about expenses
│   │   ├── ...
│   │   └── split/      # Sub-feature
│   └── settlement/     # Everything about settlements
└── pkg/                # Shared utilities
```

**Benefits of Vertical Slicing:**

1. **Cohesion** - Related code stays together
2. **Easy Navigation** - Find everything about "users" in one place
3. **Independent Teams** - Different people can work on different features
4. **Easier Refactoring** - Change one feature without touching others

## 4.2 Directory Structure Explained

```
splitwise/
│
├── cmd/                        # Application entry points
│   └── api/
│       └── main.go             # Server startup & DI wiring
│
├── internal/                   # Private application code
│   │                           # (cannot be imported by other projects)
│   │
│   ├── config/                 # Configuration management
│   │   └── config.go           # Load from environment
│   │
│   ├── database/               # Database connection
│   │   └── postgres.go         # Connection factory
│   │
│   ├── user/                   # USER FEATURE
│   │   ├── model.go            # Domain model (User struct)
│   │   ├── dto.go              # Request/Response objects
│   │   ├── repository.go       # Database operations
│   │   ├── service.go          # Business logic
│   │   └── handler.go          # HTTP handlers
│   │
│   ├── group/                  # GROUP FEATURE
│   │   └── ... (same structure)
│   │
│   ├── expense/                # EXPENSE FEATURE
│   │   ├── model.go
│   │   ├── dto.go
│   │   ├── repository.go
│   │   ├── service.go
│   │   ├── handler.go
│   │   │
│   │   └── split/              # SPLIT SUB-FEATURE
│   │       ├── strategy.go     # Interface + Factory
│   │       ├── even.go         # Even split algorithm
│   │       ├── percentage.go   # Percentage split algorithm
│   │       └── exact.go        # Exact split algorithm
│   │
│   ├── settlement/             # SETTLEMENT FEATURE
│   │   └── ...
│   │
│   └── notification/           # NOTIFICATION FEATURE
│       └── ...
│
├── pkg/                        # Public shared code
│   │                           # (can be imported by other projects)
│   │
│   ├── middleware/             # HTTP middleware
│   │   └── auth.go             # Authentication
│   │
│   └── response/               # Standard API responses
│       └── json.go             # JSON helpers
│
├── migrations/                 # Database migrations
│   ├── 000001_init_schema.up.sql
│   └── 000001_init_schema.down.sql
│
├── docs/                       # Generated Swagger docs
│
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
└── README.md                   # Project documentation
```

## 4.3 The `internal` vs `pkg` Convention

| Directory   | Visibility  | Purpose                                           |
| ----------- | ----------- | ------------------------------------------------- |
| `internal/` | **Private** | Code that shouldn't be imported by other projects |
| `pkg/`      | **Public**  | Reusable code that other projects could use       |

Go enforces this! Code in `internal/` **cannot** be imported from outside your module.

---

# 5. Design Patterns Deep Dive

## 5.1 Dependency Injection (DI)

### What is Dependency Injection?

Instead of a component creating its own dependencies, they are **injected** from outside.

**Without DI (Bad):**

```go
type UserService struct {}

func (s *UserService) GetUser(id int64) (*User, error) {
    // Service creates its own database connection - TIGHT COUPLING!
    db, _ := sql.Open("postgres", "connection-string")
    defer db.Close()

    // Query database...
}
```

**Problems:**

- Can't test without a real database
- Can't change database implementation
- Connection created on every call (inefficient)

**With DI (Good):**

```go
type UserService struct {
    repo *UserRepository  // Dependency is injected
}

// Constructor receives the dependency
func NewUserService(repo *UserRepository) *UserService {
    return &UserService{repo: repo}
}

func (s *UserService) GetUser(id int64) (*User, error) {
    return s.repo.GetByID(id)  // Uses injected repository
}
```

**Benefits:**

- **Testable** - inject a mock repository for testing
- **Flexible** - swap implementations without changing service
- **Explicit** - dependencies are visible in constructor

### DI in Our Project

Look at `cmd/api/main.go`:

```go
func main() {
    // 1. Create the database connection (shared dependency)
    db, _ := database.NewPostgresConnection(cfg.DatabaseURL)

    // 2. Create repositories (depend on db)
    userRepo := user.NewRepository(db)
    groupRepo := group.NewRepository(db)
    expenseRepo := expense.NewRepository(db)

    // 3. Create services (depend on repositories)
    splitFactory := split.NewSplitStrategyFactory()

    userService := user.NewService(userRepo)
    groupService := group.NewService(groupRepo)
    expenseService := expense.NewService(expenseRepo, splitFactory)  // Multiple deps!

    // 4. Create handlers (depend on services)
    userHandler := user.NewHandler(userService)
    groupHandler := group.NewHandler(groupService)
    expenseHandler := expense.NewHandler(expenseService)

    // 5. Wire up routes
    r.Mount("/users", userHandler.Routes())
}
```

**The Dependency Graph:**

```
┌──────────┐
│ Database │
└────┬─────┘
     │
     ▼
┌────────────┐     ┌──────────────┐
│ Repository │     │ SplitFactory │
└────┬───────┘     └──────┬───────┘
     │                    │
     └────────┬───────────┘
              ▼
        ┌─────────┐
        │ Service │
        └────┬────┘
             │
             ▼
        ┌─────────┐
        │ Handler │
        └─────────┘
```

## 5.2 Repository Pattern

### What is the Repository Pattern?

A **Repository** abstracts data access, providing a collection-like interface for domain objects.

```go
type UserRepository struct {
    db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
    return &Repository{db: db}
}

// CRUD operations
func (r *Repository) Create(ctx context.Context, user *User) error { ... }
func (r *Repository) GetByID(ctx context.Context, id int64) (*User, error) { ... }
func (r *Repository) Update(ctx context.Context, user *User) error { ... }
func (r *Repository) Delete(ctx context.Context, id int64) error { ... }
func (r *Repository) List(ctx context.Context, limit, offset int) ([]*User, error) { ... }
```

**Benefits:**

- **Abstraction** - Service doesn't know about SQL
- **Testability** - Can mock the repository
- **Single Responsibility** - Only handles data access
- **Reusability** - Same repository used by multiple services

## 5.3 Factory Pattern

### What is the Factory Pattern?

A **Factory** creates objects without exposing creation logic. It returns objects of an interface type, hiding the concrete implementation.

### Factory in Our Project: `SplitStrategyFactory`

```go
// internal/expense/split/strategy.go

type Factory struct{}

func NewSplitStrategyFactory() *Factory {
    return &Factory{}
}

// Factory method - creates appropriate strategy based on type
func (f *Factory) Create(splitType SplitType) (Strategy, error) {
    switch splitType {
    case SplitTypeEven:
        return &EvenStrategy{}, nil
    case SplitTypePercentage:
        return &PercentageStrategy{}, nil
    case SplitTypeExact:
        return &ExactStrategy{}, nil
    default:
        return nil, fmt.Errorf("unknown split type: %s", splitType)
    }
}
```

**Usage:**

```go
// Service doesn't know which concrete strategy it gets
factory := split.NewSplitStrategyFactory()
strategy, err := factory.Create(split.SplitTypeEven)

// Use the strategy (works for any type!)
splits, err := strategy.Calculate(100.0, payerID, participants)
```

## 5.4 Strategy Pattern

### What is the Strategy Pattern?

The **Strategy Pattern** defines a family of algorithms, encapsulates each one, and makes them interchangeable at runtime.

### Strategy in Our Project: Split Algorithms

We have three ways to split an expense:

| Strategy   | Description          | Example               |
| ---------- | -------------------- | --------------------- |
| EVEN       | Split equally        | $90 / 3 = $30 each    |
| PERCENTAGE | Split by percentages | 50%, 30%, 20% of $100 |
| EXACT      | Exact amounts        | $50, $30, $20         |

**The Interface:**

```go
type Strategy interface {
    Calculate(totalAmount float64, payerID int64, participants []SplitInput) ([]SplitOutput, error)
    Type() SplitType
    Validate(totalAmount float64, participants []SplitInput) error
}
```

**How It's Used:**

```go
func (s *Service) CreateExpense(ctx context.Context, payerID int64, req *CreateExpenseRequest) (*ExpenseWithSplits, error) {
    // 1. Factory creates the right strategy
    strategy, err := s.splitFactory.CreateFromString(req.SplitType)
    if err != nil {
        return nil, err
    }

    // 2. Strategy calculates splits (polymorphism!)
    splitOutputs, err := strategy.Calculate(req.Amount, payerID, inputs)
    if err != nil {
        return nil, err
    }

    // 3. Save expense and splits...
}
```

**The Magic:** The service doesn't care which strategy is used. It just calls `Calculate()` and gets the right result!

---

# 6. Layer-by-Layer Breakdown

Each feature follows this structure:

```
feature/
├── model.go      → Domain entities
├── dto.go        → Request/Response objects
├── repository.go → Data access
├── service.go    → Business logic
└── handler.go    → HTTP layer
```

## 6.1 Model Layer (`model.go`)

**Purpose:** Define the core domain entities that represent your business concepts.

```go
// internal/user/model.go
package user

import "time"

type User struct {
    ID        int64     `json:"id"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    AvatarURL *string   `json:"avatar_url,omitempty"`  // Pointer = optional
    CreatedAt time.Time `json:"created_at"`
}
```

## 6.2 DTO Layer (`dto.go`)

**Purpose:** Define data transfer objects for API requests and responses.

```go
// internal/user/dto.go
package user

// Request DTO - what the client sends
type CreateUserRequest struct {
    Username  string  `json:"username" validate:"required,min=3,max=50"`
    Email     string  `json:"email" validate:"required,email"`
    AvatarURL *string `json:"avatar_url,omitempty"`
}

// Response DTO - what we send back
type UserResponse struct {
    ID        int64   `json:"id"`
    Username  string  `json:"username"`
    Email     string  `json:"email"`
    AvatarURL *string `json:"avatar_url,omitempty"`
    CreatedAt string  `json:"created_at"`  // Formatted string
}

// Converter method
func (u *User) ToResponse() *UserResponse {
    return &UserResponse{
        ID:        u.ID,
        Username:  u.Username,
        Email:     u.Email,
        AvatarURL: u.AvatarURL,
        CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
    }
}
```

## 6.3 Repository Layer (`repository.go`)

**Purpose:** Handle all database operations.

```go
// internal/user/repository.go
type Repository struct {
    db *sql.DB  // Injected dependency
}

func NewRepository(db *sql.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*User, error) {
    query := `SELECT id, username, email, avatar_url, created_at FROM users WHERE id = $1`

    user := &User{}
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &user.ID, &user.Username, &user.Email, &user.AvatarURL, &user.CreatedAt,
    )

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, nil  // Not found
        }
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return user, nil
}
```

## 6.4 Service Layer (`service.go`)

**Purpose:** Implement business logic and validation.

```go
// internal/user/service.go
var (
    ErrUserNotFound      = errors.New("user not found")
    ErrEmailAlreadyInUse = errors.New("email already in use")
)

type Service struct {
    repo *Repository  // Injected dependency
}

func NewService(repo *Repository) *Service {
    return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req *CreateUserRequest) (*User, error) {
    // Business rule: email must be unique
    existing, _ := s.repo.GetByEmail(ctx, req.Email)
    if existing != nil {
        return nil, ErrEmailAlreadyInUse
    }

    return s.repo.Create(ctx, req)
}
```

## 6.5 Handler Layer (`handler.go`)

**Purpose:** Handle HTTP requests and responses.

```go
// internal/user/handler.go
type Handler struct {
    service *Service  // Injected dependency
}

func NewHandler(service *Service) *Handler {
    return &Handler{service: service}
}

func (h *Handler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Post("/", h.Create)
    r.Get("/", h.List)
    r.Get("/{id}", h.GetByID)
    return r
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.BadRequest(w, "Invalid request body")
        return
    }

    user, err := h.service.Create(r.Context(), &req)
    if err != nil {
        if errors.Is(err, ErrEmailAlreadyInUse) {
            response.Conflict(w, err.Error())
            return
        }
        response.InternalError(w, "Failed to create user")
        return
    }

    response.JSON(w, http.StatusCreated, user.ToResponse())
}
```

---

# 7. Database Schema

## Entity Relationship Diagram

```
┌──────────┐         ┌─────────────────┐         ┌──────────────┐
│  USERS   │────────<│  GROUP_MEMBERS  │>────────│    GROUPS    │
└────┬─────┘         └─────────────────┘         └──────────────┘
     │
     │ pays                    is owed by
     ▼                              │
┌──────────┐         ┌──────────────┴───┐        ┌──────────────┐
│ EXPENSES │────────<│     SPLITS       │>───────│ SETTLEMENTS  │
└──────────┘         └──────────────────┘        └──────────────┘
```

## Key Tables

### Users

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    avatar_url VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Expenses

```sql
CREATE TABLE expenses (
    id SERIAL PRIMARY KEY,
    group_id INTEGER NOT NULL REFERENCES groups(id),
    payer_id INTEGER NOT NULL REFERENCES users(id),
    description VARCHAR(255) NOT NULL,
    amount DECIMAL(10,2) NOT NULL CHECK (amount > 0),
    split_type VARCHAR(20) NOT NULL DEFAULT 'EVEN',
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Splits

```sql
CREATE TABLE splits (
    id SERIAL PRIMARY KEY,
    expense_id INTEGER NOT NULL REFERENCES expenses(id),
    borrower_id INTEGER NOT NULL REFERENCES users(id),
    amount_owed DECIMAL(10,2) NOT NULL,
    status split_status DEFAULT 'PENDING',
    settlement_id INTEGER REFERENCES settlements(id),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

---

# 8. API Endpoints Reference

## Base URL: `http://localhost:8080/api/v1`

| Method          | Endpoint                        | Description       |
| --------------- | ------------------------------- | ----------------- |
| **Users**       |                                 |                   |
| POST            | `/users`                        | Create user       |
| GET             | `/users`                        | List users        |
| GET             | `/users/{id}`                   | Get user          |
| **Groups**      |                                 |                   |
| POST            | `/groups`                       | Create group      |
| GET             | `/groups/{id}`                  | Get group         |
| POST            | `/groups/{id}/members`          | Add member        |
| POST            | `/groups/{id}/accept`           | Accept invite     |
| **Expenses**    |                                 |                   |
| POST            | `/expenses`                     | Create expense    |
| GET             | `/expenses/{id}`                | Get expense       |
| GET             | `/expenses/group/{groupId}`     | List by group     |
| **Splits**      |                                 |                   |
| POST            | `/expenses/splits/{id}/pay`     | Mark paid         |
| POST            | `/expenses/splits/{id}/confirm` | Confirm           |
| POST            | `/expenses/splits/{id}/dispute` | Dispute           |
| **Settlements** |                                 |                   |
| POST            | `/settlements`                  | Create settlement |
| POST            | `/settlements/{id}/pay`         | Mark paid         |
| POST            | `/settlements/{id}/confirm`     | Confirm           |
| GET             | `/settlements/balances`         | Get balances      |

---

# 9. Split Strategies

## EVEN Split

Divides equally: `$90 / 3 = $30 each`

```json
{
  "split_type": "EVEN",
  "amount": 90.0,
  "participants": [{ "user_id": 1 }, { "user_id": 2 }, { "user_id": 3 }]
}
```

## PERCENTAGE Split

By percentages (must sum to 100%):

```json
{
  "split_type": "PERCENTAGE",
  "amount": 100.0,
  "participants": [
    { "user_id": 1, "percentage": 50 },
    { "user_id": 2, "percentage": 30 },
    { "user_id": 3, "percentage": 20 }
  ]
}
```

## EXACT Split

Specific amounts (must sum to total):

```json
{
  "split_type": "EXACT",
  "amount": 75.0,
  "participants": [
    { "user_id": 1, "amount": 30.0 },
    { "user_id": 2, "amount": 25.0 },
    { "user_id": 3, "amount": 20.0 }
  ]
}
```

---

# 10. Settlement System

## Workflow

```
PENDING → PAID → CONFIRMED
            └──→ REJECTED
```

1. **PENDING**: Settlement created, splits locked
2. **PAID**: Payer sent money
3. **CONFIRMED**: Receiver confirms → all splits confirmed
4. **REJECTED**: Splits unlocked

## Auto-Calculated Roles

When creating a settlement, system determines payer/receiver:

| Net Balance  | Who Pays        | Who Receives |
| ------------ | --------------- | ------------ |
| You owe $50  | You             | Other user   |
| They owe $50 | Other user      | You          |
| $0 (mutual)  | You (initiator) | Other user   |

---

# 11. User Identification

## Test User Header

For development, use the `X-Test-User-ID` header:

```bash
curl -H "X-Test-User-ID: 2" http://localhost:8080/api/v1/groups
# Acts as user 2 (Jane)
```

## Test Users (Seed Data)

| ID  | Username   | Email            |
| --- | ---------- | ---------------- |
| 1   | john_doe   | john@example.com |
| 2   | jane_smith | jane@example.com |
| 3   | bob_wilson | bob@example.com  |

---

# 12. How to Extend This Project

## Adding a New Entity

1. Create feature directory: `mkdir internal/category`
2. Create `model.go` - domain struct
3. Create `dto.go` - request/response types
4. Create `repository.go` - database operations
5. Create `service.go` - business logic
6. Create `handler.go` - HTTP handlers
7. Wire up in `main.go`
8. Add migration

## Adding a New Split Strategy

1. Add constant in `strategy.go`:

   ```go
   SplitTypeShares SplitType = "SHARES"
   ```

2. Create `shares.go` implementing `Strategy` interface

3. Register in Factory:

   ```go
   case SplitTypeShares:
       return &SharesStrategy{}, nil
   ```

4. Update handler validation

**That's it!** The new strategy works automatically.

---

# 13. Common Go Patterns

## Constructor Functions

```go
func NewRepository(db *sql.DB) *Repository {
    return &Repository{db: db}
}
```

## Error Variables

```go
var ErrNotFound = errors.New("not found")

if errors.Is(err, ErrNotFound) { ... }
```

## Defer for Cleanup

```go
rows, _ := db.Query(...)
defer rows.Close()  // Always runs
```

## Table-Driven Tests

```go
tests := []struct {
    name     string
    input    float64
    expected float64
}{
    {"case 1", 90.0, 30.0},
    {"case 2", 100.0, 50.0},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { ... })
}
```

---

# 14. Testing

## Run Tests

```bash
go test ./...                    # All tests
go test -v ./internal/expense/   # Verbose, specific package
go test -cover ./...             # With coverage
```

---

# 15. Troubleshooting

| Error                        | Cause                  | Solution                       |
| ---------------------------- | ---------------------- | ------------------------------ |
| "package X is not in GOROOT" | Import path mismatch   | Check `go.mod` module name     |
| "sql: no rows"               | Query returned nothing | Handle `sql.ErrNoRows`         |
| "connection refused"         | DB not running         | Check Docker/connection string |

---

# 16. Glossary

| Term             | Definition                                               |
| ---------------- | -------------------------------------------------------- |
| Context          | Carries deadlines and cancellation signals               |
| DI               | Dependency Injection - passing dependencies from outside |
| DTO              | Data Transfer Object - API request/response structs      |
| Factory          | Creates objects without exposing creation logic          |
| Handler          | Processes HTTP requests                                  |
| Interface        | Contract defining method signatures                      |
| Repository       | Abstracts data access layer                              |
| Service          | Contains business logic                                  |
| Strategy         | Interchangeable algorithm pattern                        |
| Struct           | Custom type grouping related data                        |
| Vertical Slicing | Organizing code by feature                               |

---

# Congratulations!

You've learned:

- Go fundamentals
- Project architecture
- Design patterns (DI, Factory, Strategy, Repository)
- Layer organization
- Database design
- REST API development
- How to extend the project

**Happy coding!**
