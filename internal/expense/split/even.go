package split

// =============================================================================
// EVEN SPLIT STRATEGY
// Divides the expense equally among all participants
// =============================================================================

// EvenStrategy implements the Strategy interface for even splits
type EvenStrategy struct{}

// Type returns the split type identifier
func (s *EvenStrategy) Type() SplitType {
	return SplitTypeEven
}

// Validate checks if the inputs are valid for an even split
func (s *EvenStrategy) Validate(totalAmount float64, participants []SplitInput) error {
	if len(participants) == 0 {
		return ErrNoParticipants
	}
	if totalAmount < 0 {
		return ErrNegativeAmount
	}
	return nil
}

// Calculate divides the total amount evenly among all participants
// The payer is excluded from owing money (they already paid)
func (s *EvenStrategy) Calculate(totalAmount float64, payerID int64, participants []SplitInput) ([]SplitOutput, error) {
	if err := s.Validate(totalAmount, participants); err != nil {
		return nil, err
	}

	// Filter out the payer from participants who owe money
	debtors := filterPayer(payerID, participants)

	if len(debtors) == 0 {
		// Edge case: payer is the only participant, no splits needed
		return []SplitOutput{}, nil
	}

	// Calculate each person's share
	// Total participants include the payer (they paid their share)
	totalParticipants := len(participants)
	sharePerPerson := totalAmount / float64(totalParticipants)
	sharePerPerson = roundToTwoDecimals(sharePerPerson)

	// Handle rounding: distribute any remaining cents to the first debtor
	totalDistributed := sharePerPerson * float64(len(debtors))
	expectedFromDebtors := totalAmount - sharePerPerson // Total minus payer's share
	roundingDifference := roundToTwoDecimals(expectedFromDebtors - totalDistributed)

	// Build the output
	outputs := make([]SplitOutput, len(debtors))
	for i, debtor := range debtors {
		amount := sharePerPerson
		// Add rounding difference to first debtor
		if i == 0 && roundingDifference != 0 {
			amount = roundToTwoDecimals(amount + roundingDifference)
		}
		outputs[i] = SplitOutput{
			UserID:     debtor.UserID,
			AmountOwed: amount,
		}
	}

	return outputs, nil
}
