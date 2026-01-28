package split

import "math"

// =============================================================================
// EXACT SPLIT STRATEGY
// Each participant owes a specific exact amount (must sum to total)
// =============================================================================

// ExactStrategy implements the Strategy interface for exact amount splits
type ExactStrategy struct{}

// Type returns the split type identifier
func (s *ExactStrategy) Type() SplitType {
	return SplitTypeExact
}

// Validate checks if the inputs are valid for an exact split
func (s *ExactStrategy) Validate(totalAmount float64, participants []SplitInput) error {
	if len(participants) == 0 {
		return ErrNoParticipants
	}
	if totalAmount < 0 {
		return ErrNegativeAmount
	}

	// Check that all participants have amounts and they sum to total
	var totalExact float64
	for _, p := range participants {
		if p.Amount == nil {
			return ErrMissingExactAmount
		}
		if *p.Amount < 0 {
			return ErrNegativeAmount
		}
		totalExact += *p.Amount
	}

	// Allow for small floating point errors
	if math.Abs(totalExact-totalAmount) > 0.01 {
		return ErrInvalidExactAmounts
	}

	return nil
}

// Calculate returns the exact amounts specified for each participant
// The payer's amount represents their share; others owe their specified amounts
func (s *ExactStrategy) Calculate(totalAmount float64, payerID int64, participants []SplitInput) ([]SplitOutput, error) {
	if err := s.Validate(totalAmount, participants); err != nil {
		return nil, err
	}

	// Filter out the payer - they don't owe themselves
	debtors := filterPayer(payerID, participants)

	if len(debtors) == 0 {
		return []SplitOutput{}, nil
	}

	// For exact splits, we simply use the specified amounts
	outputs := make([]SplitOutput, len(debtors))
	for i, debtor := range debtors {
		outputs[i] = SplitOutput{
			UserID:     debtor.UserID,
			AmountOwed: roundToTwoDecimals(*debtor.Amount),
		}
	}

	return outputs, nil
}
