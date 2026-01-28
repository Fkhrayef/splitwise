package split

import "math"

// =============================================================================
// PERCENTAGE SPLIT STRATEGY
// Divides the expense based on specified percentages for each participant
// =============================================================================

// PercentageStrategy implements the Strategy interface for percentage-based splits
type PercentageStrategy struct{}

// Type returns the split type identifier
func (s *PercentageStrategy) Type() SplitType {
	return SplitTypePercentage
}

// Validate checks if the inputs are valid for a percentage split
func (s *PercentageStrategy) Validate(totalAmount float64, participants []SplitInput) error {
	if len(participants) == 0 {
		return ErrNoParticipants
	}
	if totalAmount < 0 {
		return ErrNegativeAmount
	}

	// Check that all participants have percentages and they sum to 100
	var totalPercentage float64
	for _, p := range participants {
		if p.Percentage == nil {
			return ErrMissingPercentage
		}
		if *p.Percentage < 0 || *p.Percentage > 100 {
			return ErrPercentageOutOfRange
		}
		totalPercentage += *p.Percentage
	}

	// Allow for small floating point errors (99.99 to 100.01)
	if math.Abs(totalPercentage-100) > 0.01 {
		return ErrInvalidPercentages
	}

	return nil
}

// Calculate divides the total amount based on each participant's percentage
// The payer's percentage represents their contribution; others owe their percentage
func (s *PercentageStrategy) Calculate(totalAmount float64, payerID int64, participants []SplitInput) ([]SplitOutput, error) {
	if err := s.Validate(totalAmount, participants); err != nil {
		return nil, err
	}

	// Filter out the payer - they don't owe themselves
	debtors := filterPayer(payerID, participants)

	if len(debtors) == 0 {
		return []SplitOutput{}, nil
	}

	// Calculate each debtor's amount based on their percentage
	outputs := make([]SplitOutput, len(debtors))
	var totalCalculated float64

	for i, debtor := range debtors {
		amount := (totalAmount * (*debtor.Percentage)) / 100
		amount = roundToTwoDecimals(amount)
		totalCalculated += amount
		outputs[i] = SplitOutput{
			UserID:     debtor.UserID,
			AmountOwed: amount,
		}
	}

	// Handle rounding: adjust last debtor to ensure total matches
	// Calculate what debtors should owe (total minus payer's share)
	payerPercentage := 0.0
	for _, p := range participants {
		if p.UserID == payerID && p.Percentage != nil {
			payerPercentage = *p.Percentage
			break
		}
	}
	expectedFromDebtors := roundToTwoDecimals((totalAmount * (100 - payerPercentage)) / 100)
	difference := roundToTwoDecimals(expectedFromDebtors - totalCalculated)

	if len(outputs) > 0 && difference != 0 {
		outputs[len(outputs)-1].AmountOwed = roundToTwoDecimals(
			outputs[len(outputs)-1].AmountOwed + difference,
		)
	}

	return outputs, nil
}
