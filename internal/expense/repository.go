package expense

import (
	"context"
	"database/sql"
	"fmt"
)

// Repository handles expense and split data persistence
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new expense repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateExpense inserts a new expense into the database
func (r *Repository) CreateExpense(ctx context.Context, payerID int64, req *CreateExpenseRequest) (*Expense, error) {
	query := `
		INSERT INTO expenses (group_id, payer_id, description, amount, image_url, split_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, group_id, payer_id, description, amount, image_url, split_type, created_at
	`

	expense := &Expense{}
	err := r.db.QueryRowContext(ctx, query,
		req.GroupID,
		payerID,
		req.Description,
		req.Amount,
		req.ImageURL,
		req.SplitType,
	).Scan(
		&expense.ID,
		&expense.GroupID,
		&expense.PayerID,
		&expense.Description,
		&expense.Amount,
		&expense.ImageURL,
		&expense.SplitType,
		&expense.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create expense: %w", err)
	}

	return expense, nil
}

// CreateSplit inserts a new split into the database
func (r *Repository) CreateSplit(ctx context.Context, expenseID, borrowerID int64, amountOwed float64) (*Split, error) {
	query := `
		INSERT INTO splits (expense_id, borrower_id, amount_owed, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, expense_id, borrower_id, amount_owed, status, dispute_reason, settlement_id, updated_at
	`

	split := &Split{}
	err := r.db.QueryRowContext(ctx, query, expenseID, borrowerID, amountOwed, SplitStatusPending).Scan(
		&split.ID,
		&split.ExpenseID,
		&split.BorrowerID,
		&split.AmountOwed,
		&split.Status,
		&split.DisputeReason,
		&split.SettlementID,
		&split.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create split: %w", err)
	}

	return split, nil
}

// GetExpenseByID retrieves an expense by its ID
func (r *Repository) GetExpenseByID(ctx context.Context, id int64) (*Expense, error) {
	query := `
		SELECT e.id, e.group_id, e.payer_id, e.description, e.amount, e.image_url, e.split_type, e.created_at, u.username
		FROM expenses e
		JOIN users u ON e.payer_id = u.id
		WHERE e.id = $1
	`

	expense := &Expense{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&expense.ID,
		&expense.GroupID,
		&expense.PayerID,
		&expense.Description,
		&expense.Amount,
		&expense.ImageURL,
		&expense.SplitType,
		&expense.CreatedAt,
		&expense.PayerUsername,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get expense: %w", err)
	}

	return expense, nil
}

// GetSplitsByExpenseID retrieves all splits for an expense
func (r *Repository) GetSplitsByExpenseID(ctx context.Context, expenseID int64) ([]*Split, error) {
	query := `
		SELECT s.id, s.expense_id, s.borrower_id, s.amount_owed, s.status, s.dispute_reason, s.settlement_id, s.updated_at, u.username
		FROM splits s
		JOIN users u ON s.borrower_id = u.id
		WHERE s.expense_id = $1
		ORDER BY s.id
	`

	rows, err := r.db.QueryContext(ctx, query, expenseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get splits: %w", err)
	}
	defer rows.Close()

	var splits []*Split
	for rows.Next() {
		split := &Split{}
		if err := rows.Scan(
			&split.ID,
			&split.ExpenseID,
			&split.BorrowerID,
			&split.AmountOwed,
			&split.Status,
			&split.DisputeReason,
			&split.SettlementID,
			&split.UpdatedAt,
			&split.BorrowerUsername,
		); err != nil {
			return nil, fmt.Errorf("failed to scan split: %w", err)
		}
		splits = append(splits, split)
	}

	return splits, nil
}

// ListExpensesByGroupID retrieves all expenses for a group
func (r *Repository) ListExpensesByGroupID(ctx context.Context, groupID int64, limit, offset int) ([]*Expense, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM expenses WHERE group_id = $1`
	if err := r.db.QueryRowContext(ctx, countQuery, groupID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count expenses: %w", err)
	}

	// Get expenses
	query := `
		SELECT e.id, e.group_id, e.payer_id, e.description, e.amount, e.image_url, e.split_type, e.created_at, u.username
		FROM expenses e
		JOIN users u ON e.payer_id = u.id
		WHERE e.group_id = $1
		ORDER BY e.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, groupID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list expenses: %w", err)
	}
	defer rows.Close()

	var expenses []*Expense
	for rows.Next() {
		expense := &Expense{}
		if err := rows.Scan(
			&expense.ID,
			&expense.GroupID,
			&expense.PayerID,
			&expense.Description,
			&expense.Amount,
			&expense.ImageURL,
			&expense.SplitType,
			&expense.CreatedAt,
			&expense.PayerUsername,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan expense: %w", err)
		}
		expenses = append(expenses, expense)
	}

	return expenses, total, nil
}

// GetSplitByID retrieves a split by its ID
func (r *Repository) GetSplitByID(ctx context.Context, id int64) (*Split, error) {
	query := `
		SELECT s.id, s.expense_id, s.borrower_id, s.amount_owed, s.status, s.dispute_reason, s.settlement_id, s.updated_at, u.username
		FROM splits s
		JOIN users u ON s.borrower_id = u.id
		WHERE s.id = $1
	`

	split := &Split{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&split.ID,
		&split.ExpenseID,
		&split.BorrowerID,
		&split.AmountOwed,
		&split.Status,
		&split.DisputeReason,
		&split.SettlementID,
		&split.UpdatedAt,
		&split.BorrowerUsername,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get split: %w", err)
	}

	return split, nil
}

// UpdateSplitStatus updates the status of a split
func (r *Repository) UpdateSplitStatus(ctx context.Context, id int64, status SplitStatus, disputeReason *string) (*Split, error) {
	query := `
		UPDATE splits
		SET status = $2, dispute_reason = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, expense_id, borrower_id, amount_owed, status, dispute_reason, settlement_id, updated_at
	`

	split := &Split{}
	err := r.db.QueryRowContext(ctx, query, id, status, disputeReason).Scan(
		&split.ID,
		&split.ExpenseID,
		&split.BorrowerID,
		&split.AmountOwed,
		&split.Status,
		&split.DisputeReason,
		&split.SettlementID,
		&split.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to update split status: %w", err)
	}

	return split, nil
}

// GetPendingSplitsBetweenUsers gets all pending/paid splits where user1 owes user2
func (r *Repository) GetPendingSplitsBetweenUsers(ctx context.Context, borrowerID, payerID int64) ([]*Split, error) {
	query := `
		SELECT s.id, s.expense_id, s.borrower_id, s.amount_owed, s.status, s.dispute_reason, s.settlement_id, s.updated_at
		FROM splits s
		JOIN expenses e ON s.expense_id = e.id
		WHERE s.borrower_id = $1 
		  AND e.payer_id = $2
		  AND s.status IN ('PENDING', 'PAID')
		  AND s.settlement_id IS NULL
		ORDER BY s.id
	`

	rows, err := r.db.QueryContext(ctx, query, borrowerID, payerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get splits: %w", err)
	}
	defer rows.Close()

	var splits []*Split
	for rows.Next() {
		split := &Split{}
		if err := rows.Scan(
			&split.ID,
			&split.ExpenseID,
			&split.BorrowerID,
			&split.AmountOwed,
			&split.Status,
			&split.DisputeReason,
			&split.SettlementID,
			&split.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan split: %w", err)
		}
		splits = append(splits, split)
	}

	return splits, nil
}

// LockSplitsToSettlement locks splits to a settlement
func (r *Repository) LockSplitsToSettlement(ctx context.Context, splitIDs []int64, settlementID int64) error {
	for _, splitID := range splitIDs {
		query := `UPDATE splits SET settlement_id = $2, updated_at = NOW() WHERE id = $1`
		_, err := r.db.ExecContext(ctx, query, splitID, settlementID)
		if err != nil {
			return fmt.Errorf("failed to lock split %d: %w", splitID, err)
		}
	}
	return nil
}

// UnlockSplitsFromSettlement removes the settlement lock from splits
func (r *Repository) UnlockSplitsFromSettlement(ctx context.Context, settlementID int64) error {
	query := `UPDATE splits SET settlement_id = NULL, updated_at = NOW() WHERE settlement_id = $1`
	_, err := r.db.ExecContext(ctx, query, settlementID)
	if err != nil {
		return fmt.Errorf("failed to unlock splits: %w", err)
	}
	return nil
}

// ConfirmSplitsBySettlement marks all splits in a settlement as confirmed
func (r *Repository) ConfirmSplitsBySettlement(ctx context.Context, settlementID int64) error {
	query := `UPDATE splits SET status = $2, updated_at = NOW() WHERE settlement_id = $1`
	_, err := r.db.ExecContext(ctx, query, settlementID, SplitStatusConfirmed)
	if err != nil {
		return fmt.Errorf("failed to confirm splits: %w", err)
	}
	return nil
}

// DeleteExpense deletes an expense and its splits
func (r *Repository) DeleteExpense(ctx context.Context, id int64) error {
	// Delete splits first (foreign key constraint)
	_, err := r.db.ExecContext(ctx, `DELETE FROM splits WHERE expense_id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete splits: %w", err)
	}

	// Delete expense
	result, err := r.db.ExecContext(ctx, `DELETE FROM expenses WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete expense: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("expense not found")
	}

	return nil
}
