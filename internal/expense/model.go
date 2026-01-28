package expense

import (
	"time"

	"github.com/fkhayef/splitwise/internal/expense/split"
)

// SplitStatus represents the status of a split
type SplitStatus string

const (
	SplitStatusPending   SplitStatus = "PENDING"
	SplitStatusPaid      SplitStatus = "PAID"
	SplitStatusConfirmed SplitStatus = "CONFIRMED"
	SplitStatusDisputed  SplitStatus = "DISPUTED"
)

// Expense represents an expense in the system
type Expense struct {
	ID          int64     `json:"id"`
	GroupID     int64     `json:"group_id"`
	PayerID     int64     `json:"payer_id"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	ImageURL    *string   `json:"image_url,omitempty"`
	SplitType   string    `json:"split_type"` // EVEN, PERCENTAGE, EXACT
	CreatedAt   time.Time `json:"created_at"`

	// Populated via JOIN
	PayerUsername string `json:"payer_username,omitempty"`
}

// Split represents an individual debt from an expense
type Split struct {
	ID            int64       `json:"id"`
	ExpenseID     int64       `json:"expense_id"`
	BorrowerID    int64       `json:"borrower_id"`
	AmountOwed    float64     `json:"amount_owed"`
	Status        SplitStatus `json:"status"`
	DisputeReason *string     `json:"dispute_reason,omitempty"`
	SettlementID  *int64      `json:"settlement_id,omitempty"` // Optional: locked to settlement
	UpdatedAt     time.Time   `json:"updated_at"`

	// Populated via JOIN
	BorrowerUsername string `json:"borrower_username,omitempty"`
}

// ExpenseWithSplits combines an expense with its calculated splits
type ExpenseWithSplits struct {
	Expense *Expense
	Splits  []*Split
}

// SplitParticipant is used when creating an expense with splits
type SplitParticipant struct {
	UserID     int64                `json:"user_id"`
	Percentage *float64             `json:"percentage,omitempty"` // For PERCENTAGE split
	Amount     *float64             `json:"amount,omitempty"`     // For EXACT split
}

// ToSplitInput converts to the split package's input type
func (p *SplitParticipant) ToSplitInput() split.SplitInput {
	return split.SplitInput{
		UserID:     p.UserID,
		Percentage: p.Percentage,
		Amount:     p.Amount,
	}
}
