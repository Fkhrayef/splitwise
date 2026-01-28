package expense

import (
	"context"
	"errors"

	"github.com/fkhayef/splitwise/internal/expense/split"
)

// Common errors
var (
	ErrExpenseNotFound      = errors.New("expense not found")
	ErrSplitNotFound        = errors.New("split not found")
	ErrSplitLocked          = errors.New("split is locked to a settlement")
	ErrNotBorrower          = errors.New("only the borrower can mark as paid")
	ErrNotPayer             = errors.New("only the payer can confirm payment")
	ErrInvalidStatusChange  = errors.New("invalid status change")
	ErrCannotDeleteExpense  = errors.New("cannot delete expense with paid/confirmed splits")
)

// Service handles expense business logic
type Service struct {
	repo         *Repository
	splitFactory *split.Factory // Factory pattern for creating split strategies
}

// NewService creates a new expense service with dependencies injected
func NewService(repo *Repository, splitFactory *split.Factory) *Service {
	return &Service{
		repo:         repo,
		splitFactory: splitFactory,
	}
}

// CreateExpense creates a new expense and calculates splits using the appropriate strategy
func (s *Service) CreateExpense(ctx context.Context, payerID int64, req *CreateExpenseRequest) (*ExpenseWithSplits, error) {
	// Use FACTORY PATTERN to get the appropriate split strategy
	strategy, err := s.splitFactory.CreateFromString(req.SplitType)
	if err != nil {
		return nil, err
	}

	// Convert participants to split inputs
	inputs := make([]split.SplitInput, len(req.Participants))
	for i, p := range req.Participants {
		inputs[i] = p.ToSplitInput()
	}

	// Use STRATEGY PATTERN - calculate splits using the selected strategy
	splitOutputs, err := strategy.Calculate(req.Amount, payerID, inputs)
	if err != nil {
		return nil, err
	}

	// Create the expense
	expense, err := s.repo.CreateExpense(ctx, payerID, req)
	if err != nil {
		return nil, err
	}

	// Create the splits
	splits := make([]*Split, len(splitOutputs))
	for i, output := range splitOutputs {
		split, err := s.repo.CreateSplit(ctx, expense.ID, output.UserID, output.AmountOwed)
		if err != nil {
			// TODO: Should rollback expense creation in a transaction
			return nil, err
		}
		splits[i] = split
	}

	return &ExpenseWithSplits{
		Expense: expense,
		Splits:  splits,
	}, nil
}

// GetExpenseByID retrieves an expense with its splits
func (s *Service) GetExpenseByID(ctx context.Context, id int64) (*ExpenseWithSplits, error) {
	expense, err := s.repo.GetExpenseByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, ErrExpenseNotFound
	}

	splits, err := s.repo.GetSplitsByExpenseID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &ExpenseWithSplits{
		Expense: expense,
		Splits:  splits,
	}, nil
}

// ListExpensesByGroupID retrieves expenses for a group
func (s *Service) ListExpensesByGroupID(ctx context.Context, groupID int64, page, perPage int) ([]*Expense, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage
	return s.repo.ListExpensesByGroupID(ctx, groupID, perPage, offset)
}

// MarkSplitAsPaid allows the borrower to mark their split as paid
func (s *Service) MarkSplitAsPaid(ctx context.Context, splitID, borrowerID int64) (*Split, error) {
	split, err := s.repo.GetSplitByID(ctx, splitID)
	if err != nil {
		return nil, err
	}
	if split == nil {
		return nil, ErrSplitNotFound
	}

	// Check if the user is the borrower
	if split.BorrowerID != borrowerID {
		return nil, ErrNotBorrower
	}

	// Check if split is locked to a settlement
	if split.SettlementID != nil {
		return nil, ErrSplitLocked
	}

	// Can only mark as paid from PENDING status
	if split.Status != SplitStatusPending {
		return nil, ErrInvalidStatusChange
	}

	return s.repo.UpdateSplitStatus(ctx, splitID, SplitStatusPaid, nil)
}

// ConfirmSplitPayment allows the payer to confirm they received the payment
func (s *Service) ConfirmSplitPayment(ctx context.Context, splitID, payerID int64) (*Split, error) {
	split, err := s.repo.GetSplitByID(ctx, splitID)
	if err != nil {
		return nil, err
	}
	if split == nil {
		return nil, ErrSplitNotFound
	}

	// Get the expense to check if user is the payer
	expense, err := s.repo.GetExpenseByID(ctx, split.ExpenseID)
	if err != nil {
		return nil, err
	}
	if expense.PayerID != payerID {
		return nil, ErrNotPayer
	}

	// Check if split is locked to a settlement
	if split.SettlementID != nil {
		return nil, ErrSplitLocked
	}

	// Can only confirm from PAID status
	if split.Status != SplitStatusPaid {
		return nil, ErrInvalidStatusChange
	}

	return s.repo.UpdateSplitStatus(ctx, splitID, SplitStatusConfirmed, nil)
}

// DisputeSplit allows the borrower to dispute a split
func (s *Service) DisputeSplit(ctx context.Context, splitID, borrowerID int64, reason string) (*Split, error) {
	split, err := s.repo.GetSplitByID(ctx, splitID)
	if err != nil {
		return nil, err
	}
	if split == nil {
		return nil, ErrSplitNotFound
	}

	// Check if the user is the borrower
	if split.BorrowerID != borrowerID {
		return nil, ErrNotBorrower
	}

	// Can dispute from PENDING or PAID status
	if split.Status != SplitStatusPending && split.Status != SplitStatusPaid {
		return nil, ErrInvalidStatusChange
	}

	return s.repo.UpdateSplitStatus(ctx, splitID, SplitStatusDisputed, &reason)
}

// DeleteExpense deletes an expense if no splits are paid/confirmed
func (s *Service) DeleteExpense(ctx context.Context, id, userID int64) error {
	expense, err := s.repo.GetExpenseByID(ctx, id)
	if err != nil {
		return err
	}
	if expense == nil {
		return ErrExpenseNotFound
	}

	// Only the payer can delete
	if expense.PayerID != userID {
		return ErrNotPayer
	}

	// Check if any splits are paid or confirmed
	splits, err := s.repo.GetSplitsByExpenseID(ctx, id)
	if err != nil {
		return err
	}
	for _, split := range splits {
		if split.Status == SplitStatusPaid || split.Status == SplitStatusConfirmed {
			return ErrCannotDeleteExpense
		}
	}

	return s.repo.DeleteExpense(ctx, id)
}
