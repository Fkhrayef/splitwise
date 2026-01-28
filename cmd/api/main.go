package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/fkhayef/splitwise/internal/config"
	mw "github.com/fkhayef/splitwise/pkg/middleware"
	"github.com/fkhayef/splitwise/internal/database"
	"github.com/fkhayef/splitwise/internal/expense"
	expensesplit "github.com/fkhayef/splitwise/internal/expense/split"
	"github.com/fkhayef/splitwise/internal/group"
	"github.com/fkhayef/splitwise/internal/notification"
	"github.com/fkhayef/splitwise/internal/settlement"
	"github.com/fkhayef/splitwise/internal/user"

	_ "github.com/fkhayef/splitwise/docs" // Swagger docs
)

// @title           Splitwise API
// @version         1.0
// @description     A Splitwise-like expense splitting API demonstrating DI, Factory, and Strategy patterns.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@splitwise.local

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer {token}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database connection
	db, err := database.NewPostgresConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to database successfully")

	// ============================================
	// DEPENDENCY INJECTION - Wiring up all layers
	// ============================================

	// Split Strategy Factory (Factory Pattern)
	splitFactory := expensesplit.NewSplitStrategyFactory()

	// User feature
	userRepo := user.NewRepository(db)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)

	// Group feature
	groupRepo := group.NewRepository(db)
	groupService := group.NewService(groupRepo)
	groupHandler := group.NewHandler(groupService)

	// Expense feature (with split factory injected)
	expenseRepo := expense.NewRepository(db)
	expenseService := expense.NewService(expenseRepo, splitFactory)
	expenseHandler := expense.NewHandler(expenseService)

	// Settlement feature
	settlementRepo := settlement.NewRepository(db)
	settlementService := settlement.NewService(settlementRepo, expenseRepo)
	settlementHandler := settlement.NewHandler(settlementService)

	// Notification feature
	notificationRepo := notification.NewRepository(db)
	notificationService := notification.NewService(notificationRepo)
	notificationHandler := notification.NewHandler(notificationService)

	// ============================================
	// ROUTER SETUP
	// ============================================

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(mw.TestUserMiddleware) // DEV ONLY: Allows X-Test-User-ID header

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Mount feature routers
		r.Mount("/users", userHandler.Routes())
		r.Mount("/groups", groupHandler.Routes())
		r.Mount("/expenses", expenseHandler.Routes())
		r.Mount("/settlements", settlementHandler.Routes())
		r.Mount("/notifications", notificationHandler.Routes())
	})

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
