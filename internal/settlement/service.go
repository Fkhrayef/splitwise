package settlement

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/fkhayef/splitwise/internal/expense"
)

// Common errors
var (
	ErrSettlementNotFound  = errors.New("settlement not found")
	ErrAlreadySettled      = errors.New("already settled up - no pending debts")
	ErrNotPayer            = errors.New("only the payer can mark as paid")
	ErrNotReceiver         = errors.New("only the receiver can confirm/reject")
	ErrInvalidStatusChange = errors.New("invalid status change")
	ErrCannotSettleSelf    = errors.New("cannot create settlement with yourself")
)

// Service handles settlement business logic
type Service struct {
	repo        *Repository
	expenseRepo *expense.Repository
}

// NewService creates a new settlement service
func NewService(repo *Repository, expenseRepo *expense.Repository) *Service {
	return &Service{
		repo:        repo,
		expenseRepo: expenseRepo,
	}
}

// CreateSettlement creates a new bulk settlement between two users
// Anyone can initiate - system determines payer/receiver based on net balance
// Even $0 settlements are valid (just need confirmation to clear pending debts)
func (s *Service) CreateSettlement(ctx context.Context, initiatorID int64, req *CreateSettlementRequest) (*Settlement, error) {
	otherUserID := req.OtherUserID
	
	if initiatorID == otherUserID {
		return nil, ErrCannotSettleSelf
	}

	// Calculate net balance from initiator's perspective
	// Positive = initiator owes other user
	// Negative = other user owes initiator
	netBalance, err := s.repo.GetNetBalanceBetweenUsers(ctx, initiatorID, otherUserID)
	if err != nil {
		return nil, err
	}

	// Determine payer and receiver based on who owes whom
	var payerID, receiverID int64
	var amount float64

	if netBalance > 0 {
		// Initiator owes the other user
		payerID = initiatorID
		receiverID = otherUserID
		amount = math.Round(netBalance*100) / 100
	} else if netBalance < 0 {
		// Other user owes the initiator
		payerID = otherUserID
		receiverID = initiatorID
		amount = math.Round(-netBalance*100) / 100 // Make positive
	} else {
		// Net is zero - check if there are any pending splits at all
		// (there might be mutual debts that cancel out)
		splits1, _ := s.expenseRepo.GetPendingSplitsBetweenUsers(ctx, initiatorID, otherUserID)
		splits2, _ := s.expenseRepo.GetPendingSplitsBetweenUsers(ctx, otherUserID, initiatorID)
		
		if len(splits1) == 0 && len(splits2) == 0 {
			return nil, ErrAlreadySettled
		}
		
		// Zero-amount settlement: initiator requests, other confirms
		payerID = initiatorID
		receiverID = otherUserID
		amount = 0
	}

	// Get all pending splits in BOTH directions to lock them
	splitsInitiatorOwes, err := s.expenseRepo.GetPendingSplitsBetweenUsers(ctx, initiatorID, otherUserID)
	if err != nil {
		return nil, err
	}
	splitsOtherOwes, err := s.expenseRepo.GetPendingSplitsBetweenUsers(ctx, otherUserID, initiatorID)
	if err != nil {
		return nil, err
	}

	// Create the settlement
	settlement, err := s.repo.Create(ctx, payerID, receiverID, amount)
	if err != nil {
		return nil, err
	}

	// Lock ALL splits between these users (both directions)
	var allSplitIDs []int64
	for _, split := range splitsInitiatorOwes {
		allSplitIDs = append(allSplitIDs, split.ID)
	}
	for _, split := range splitsOtherOwes {
		allSplitIDs = append(allSplitIDs, split.ID)
	}

	if len(allSplitIDs) > 0 {
		if err := s.expenseRepo.LockSplitsToSettlement(ctx, allSplitIDs, settlement.ID); err != nil {
			// TODO: Should rollback settlement creation in a transaction
			return nil, err
		}
	}

	return settlement, nil
}

// GetByID retrieves a settlement by its ID
func (s *Service) GetByID(ctx context.Context, id int64) (*Settlement, error) {
	settlement, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if settlement == nil {
		return nil, ErrSettlementNotFound
	}
	return settlement, nil
}

// ListByUserID retrieves all settlements for a user
func (s *Service) ListByUserID(ctx context.Context, userID int64, page, perPage int) ([]*Settlement, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage
	return s.repo.ListByUserID(ctx, userID, perPage, offset)
}

// MarkAsPaid allows the payer to mark the settlement as paid
func (s *Service) MarkAsPaid(ctx context.Context, settlementID, userID int64) (*Settlement, error) {
	settlement, err := s.repo.GetByID(ctx, settlementID)
	if err != nil {
		return nil, err
	}
	if settlement == nil {
		return nil, ErrSettlementNotFound
	}

	// Only the payer can mark as paid
	if settlement.PayerID != userID {
		return nil, ErrNotPayer
	}

	// Can only mark as paid from PENDING status
	if settlement.Status != SettlementStatusPending {
		return nil, ErrInvalidStatusChange
	}

	return s.repo.UpdateStatus(ctx, settlementID, SettlementStatusPaid)
}

// Confirm allows the receiver to confirm they received the payment
func (s *Service) Confirm(ctx context.Context, settlementID, userID int64) (*Settlement, error) {
	settlement, err := s.repo.GetByID(ctx, settlementID)
	if err != nil {
		return nil, err
	}
	if settlement == nil {
		return nil, ErrSettlementNotFound
	}

	// Only the receiver can confirm
	if settlement.ReceiverID != userID {
		return nil, ErrNotReceiver
	}

	// Can only confirm from PAID status
	if settlement.Status != SettlementStatusPaid {
		return nil, ErrInvalidStatusChange
	}

	// Update settlement status
	settlement, err = s.repo.UpdateStatus(ctx, settlementID, SettlementStatusConfirmed)
	if err != nil {
		return nil, err
	}

	// Mark all locked splits as confirmed
	if err := s.expenseRepo.ConfirmSplitsBySettlement(ctx, settlementID); err != nil {
		return nil, err
	}

	return settlement, nil
}

// Reject allows the receiver to reject the settlement (they didn't receive payment)
func (s *Service) Reject(ctx context.Context, settlementID, userID int64) (*Settlement, error) {
	settlement, err := s.repo.GetByID(ctx, settlementID)
	if err != nil {
		return nil, err
	}
	if settlement == nil {
		return nil, ErrSettlementNotFound
	}

	// Only the receiver can reject
	if settlement.ReceiverID != userID {
		return nil, ErrNotReceiver
	}

	// Can reject from PENDING or PAID status
	if settlement.Status != SettlementStatusPending && settlement.Status != SettlementStatusPaid {
		return nil, ErrInvalidStatusChange
	}

	// Update settlement status
	settlement, err = s.repo.UpdateStatus(ctx, settlementID, SettlementStatusRejected)
	if err != nil {
		return nil, err
	}

	// Unlock all splits from this settlement
	if err := s.expenseRepo.UnlockSplitsFromSettlement(ctx, settlementID); err != nil {
		return nil, err
	}

	return settlement, nil
}

// GetNetBalances returns all net balances for a user
func (s *Service) GetNetBalances(ctx context.Context, userID int64) ([]*NetBalanceResponse, error) {
	balances, err := s.repo.GetNetBalancesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]*NetBalanceResponse, len(balances))
	for i, b := range balances {
		var message string
		if b.Amount > 0 {
			message = fmt.Sprintf("You owe %s $%.2f", b.Username, b.Amount)
		} else {
			message = fmt.Sprintf("%s owes you $%.2f", b.Username, -b.Amount)
		}
		responses[i] = &NetBalanceResponse{
			UserID:   b.UserID,
			Username: b.Username,
			Amount:   b.Amount,
			Message:  message,
		}
	}

	return responses, nil
}

// GetNetBalanceWithUser returns the net balance with a specific user
func (s *Service) GetNetBalanceWithUser(ctx context.Context, userID, otherUserID int64, otherUsername string) (*NetBalanceResponse, error) {
	amount, err := s.repo.GetNetBalanceBetweenUsers(ctx, userID, otherUserID)
	if err != nil {
		return nil, err
	}

	var message string
	if amount > 0 {
		message = fmt.Sprintf("You owe %s $%.2f", otherUsername, amount)
	} else if amount < 0 {
		message = fmt.Sprintf("%s owes you $%.2f", otherUsername, -amount)
	} else {
		message = fmt.Sprintf("You and %s are settled up", otherUsername)
	}

	return &NetBalanceResponse{
		UserID:   otherUserID,
		Username: otherUsername,
		Amount:   amount,
		Message:  message,
	}, nil
}
