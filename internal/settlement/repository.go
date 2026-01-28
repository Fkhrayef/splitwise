package settlement

import (
	"context"
	"database/sql"
	"fmt"
)

// Repository handles settlement data persistence
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new settlement repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new settlement into the database
func (r *Repository) Create(ctx context.Context, payerID, receiverID int64, amount float64) (*Settlement, error) {
	query := `
		INSERT INTO settlements (payer_id, receiver_id, amount, currency_code, status)
		VALUES ($1, $2, $3, 'SAR', $4)
		RETURNING id, payer_id, receiver_id, amount, currency_code, status, created_at
	`

	settlement := &Settlement{}
	err := r.db.QueryRowContext(ctx, query, payerID, receiverID, amount, SettlementStatusPending).Scan(
		&settlement.ID,
		&settlement.PayerID,
		&settlement.ReceiverID,
		&settlement.Amount,
		&settlement.CurrencyCode,
		&settlement.Status,
		&settlement.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create settlement: %w", err)
	}

	return settlement, nil
}

// GetByID retrieves a settlement by its ID
func (r *Repository) GetByID(ctx context.Context, id int64) (*Settlement, error) {
	query := `
		SELECT s.id, s.payer_id, s.receiver_id, s.amount, s.currency_code, s.status, s.created_at,
		       p.username as payer_username, recv.username as receiver_username
		FROM settlements s
		JOIN users p ON s.payer_id = p.id
		JOIN users recv ON s.receiver_id = recv.id
		WHERE s.id = $1
	`

	settlement := &Settlement{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&settlement.ID,
		&settlement.PayerID,
		&settlement.ReceiverID,
		&settlement.Amount,
		&settlement.CurrencyCode,
		&settlement.Status,
		&settlement.CreatedAt,
		&settlement.PayerUsername,
		&settlement.ReceiverUsername,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get settlement: %w", err)
	}

	return settlement, nil
}

// ListByUserID retrieves all settlements involving a user
func (r *Repository) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]*Settlement, int, error) {
	// Get total count
	var total int
	countQuery := `
		SELECT COUNT(*) FROM settlements 
		WHERE payer_id = $1 OR receiver_id = $1
	`
	if err := r.db.QueryRowContext(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count settlements: %w", err)
	}

	// Get settlements
	query := `
		SELECT s.id, s.payer_id, s.receiver_id, s.amount, s.currency_code, s.status, s.created_at,
		       p.username as payer_username, recv.username as receiver_username
		FROM settlements s
		JOIN users p ON s.payer_id = p.id
		JOIN users recv ON s.receiver_id = recv.id
		WHERE s.payer_id = $1 OR s.receiver_id = $1
		ORDER BY s.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list settlements: %w", err)
	}
	defer rows.Close()

	var settlements []*Settlement
	for rows.Next() {
		settlement := &Settlement{}
		if err := rows.Scan(
			&settlement.ID,
			&settlement.PayerID,
			&settlement.ReceiverID,
			&settlement.Amount,
			&settlement.CurrencyCode,
			&settlement.Status,
			&settlement.CreatedAt,
			&settlement.PayerUsername,
			&settlement.ReceiverUsername,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan settlement: %w", err)
		}
		settlements = append(settlements, settlement)
	}

	return settlements, total, nil
}

// UpdateStatus updates the status of a settlement
func (r *Repository) UpdateStatus(ctx context.Context, id int64, status SettlementStatus) (*Settlement, error) {
	query := `
		UPDATE settlements
		SET status = $2
		WHERE id = $1
		RETURNING id, payer_id, receiver_id, amount, currency_code, status, created_at
	`

	settlement := &Settlement{}
	err := r.db.QueryRowContext(ctx, query, id, status).Scan(
		&settlement.ID,
		&settlement.PayerID,
		&settlement.ReceiverID,
		&settlement.Amount,
		&settlement.CurrencyCode,
		&settlement.Status,
		&settlement.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to update settlement status: %w", err)
	}

	return settlement, nil
}

// GetNetBalancesForUser calculates net balances with all other users
func (r *Repository) GetNetBalancesForUser(ctx context.Context, userID int64) ([]*NetBalance, error) {
	// This query calculates the net balance:
	// Positive = user owes them (they paid for user)
	// Negative = they owe user (user paid for them)
	query := `
		WITH 
		-- What user owes others (from expenses where others paid)
		user_owes AS (
			SELECT e.payer_id as other_user_id, SUM(s.amount_owed) as amount
			FROM splits s
			JOIN expenses e ON s.expense_id = e.id
			WHERE s.borrower_id = $1 
			  AND s.status IN ('PENDING', 'PAID')
			  AND s.settlement_id IS NULL
			GROUP BY e.payer_id
		),
		-- What others owe user (from expenses where user paid)
		others_owe AS (
			SELECT s.borrower_id as other_user_id, SUM(s.amount_owed) as amount
			FROM splits s
			JOIN expenses e ON s.expense_id = e.id
			WHERE e.payer_id = $1 
			  AND s.status IN ('PENDING', 'PAID')
			  AND s.settlement_id IS NULL
			GROUP BY s.borrower_id
		),
		-- Combine and calculate net
		net_balances AS (
			SELECT 
				COALESCE(uo.other_user_id, oo.other_user_id) as other_user_id,
				COALESCE(uo.amount, 0) - COALESCE(oo.amount, 0) as net_amount
			FROM user_owes uo
			FULL OUTER JOIN others_owe oo ON uo.other_user_id = oo.other_user_id
		)
		SELECT nb.other_user_id, u.username, nb.net_amount
		FROM net_balances nb
		JOIN users u ON nb.other_user_id = u.id
		WHERE nb.net_amount != 0
		ORDER BY ABS(nb.net_amount) DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get net balances: %w", err)
	}
	defer rows.Close()

	var balances []*NetBalance
	for rows.Next() {
		balance := &NetBalance{}
		if err := rows.Scan(&balance.UserID, &balance.Username, &balance.Amount); err != nil {
			return nil, fmt.Errorf("failed to scan net balance: %w", err)
		}
		balances = append(balances, balance)
	}

	return balances, nil
}

// GetNetBalanceBetweenUsers calculates the net balance between two specific users
func (r *Repository) GetNetBalanceBetweenUsers(ctx context.Context, userID, otherUserID int64) (float64, error) {
	query := `
		WITH 
		user_owes AS (
			SELECT COALESCE(SUM(s.amount_owed), 0) as amount
			FROM splits s
			JOIN expenses e ON s.expense_id = e.id
			WHERE s.borrower_id = $1 
			  AND e.payer_id = $2
			  AND s.status IN ('PENDING', 'PAID')
			  AND s.settlement_id IS NULL
		),
		other_owes AS (
			SELECT COALESCE(SUM(s.amount_owed), 0) as amount
			FROM splits s
			JOIN expenses e ON s.expense_id = e.id
			WHERE s.borrower_id = $2 
			  AND e.payer_id = $1
			  AND s.status IN ('PENDING', 'PAID')
			  AND s.settlement_id IS NULL
		)
		SELECT (SELECT amount FROM user_owes) - (SELECT amount FROM other_owes)
	`

	var netAmount float64
	if err := r.db.QueryRowContext(ctx, query, userID, otherUserID).Scan(&netAmount); err != nil {
		return 0, fmt.Errorf("failed to get net balance: %w", err)
	}

	return netAmount, nil
}
