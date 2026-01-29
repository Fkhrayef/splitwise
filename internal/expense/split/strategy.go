package split

import (
	"errors"
	"fmt"
	"math"
)

// SplitType defines the type of split strategy
type SplitType string

const (
	SplitTypeEven       SplitType = "EVEN"
	SplitTypePercentage SplitType = "PERCENTAGE"
	SplitTypeExact      SplitType = "EXACT"
)

// SplitInput represents a participant in a split with optional values
type SplitInput struct {
	UserID     int64    `json:"user_id"`
	Percentage *float64 `json:"percentage,omitempty"` // For PERCENTAGE split
	Amount     *float64 `json:"amount,omitempty"`     // For EXACT split
}

// SplitOutput represents the calculated split for a single participant
type SplitOutput struct {
	UserID     int64   `json:"user_id"`
	AmountOwed float64 `json:"amount_owed"`
}

// Strategy is the interface that all split strategies must implement
type Strategy interface {
	// Calculate computes the split amounts for all participants
	Calculate(totalAmount float64, payerID int64, participants []SplitInput) ([]SplitOutput, error)

	// Type returns the type identifier for this strategy
	Type() SplitType

	// Validate checks if the inputs are valid for this strategy
	Validate(totalAmount float64, participants []SplitInput) error
}

// Factory creates split strategies based on the requested type
type Factory struct{}

// NewSplitStrategyFactory creates a new factory instance
func NewSplitStrategyFactory() *Factory {
	return &Factory{}
}

// Create returns the appropriate strategy implementation based on the type
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

// CreateFromString creates a strategy from a string type (useful for API requests)
func (f *Factory) CreateFromString(splitType string) (Strategy, error) {
	return f.Create(SplitType(splitType))
}

var (
	ErrNoParticipants       = errors.New("at least one participant is required")
	ErrInvalidPercentages   = errors.New("percentages must sum to 100")
	ErrInvalidExactAmounts  = errors.New("exact amounts must sum to total amount")
	ErrNegativeAmount       = errors.New("amounts cannot be negative")
	ErrMissingPercentage    = errors.New("percentage value required for all participants")
	ErrMissingExactAmount   = errors.New("exact amount required for all participants")
	ErrPercentageOutOfRange = errors.New("percentage must be between 0 and 100")
)

// roundToTwoDecimals rounds a float to 2 decimal places
func roundToTwoDecimals(value float64) float64 {
	return math.Round(value*100) / 100
}

// filterPayer removes the payer from participants (they don't owe themselves)
func filterPayer(payerID int64, participants []SplitInput) []SplitInput {
	filtered := make([]SplitInput, 0, len(participants))
	for _, p := range participants {
		if p.UserID != payerID {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
