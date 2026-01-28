package expense

// CreateExpenseRequest represents the request to create an expense
type CreateExpenseRequest struct {
	GroupID      int64               `json:"group_id" validate:"required"`
	Description  string              `json:"description" validate:"required,min=1,max=255"`
	Amount       float64             `json:"amount" validate:"required,gt=0"`
	ImageURL     *string             `json:"image_url,omitempty"`
	SplitType    string              `json:"split_type" validate:"required,oneof=EVEN PERCENTAGE EXACT"`
	Participants []*SplitParticipant `json:"participants" validate:"required,min=1"`
}

// UpdateExpenseRequest represents the request to update an expense
type UpdateExpenseRequest struct {
	Description *string  `json:"description,omitempty" validate:"omitempty,min=1,max=255"`
	ImageURL    *string  `json:"image_url,omitempty"`
}

// MarkSplitPaidRequest represents the request to mark a split as paid
type MarkSplitPaidRequest struct {
	// No body needed - uses authenticated user
}

// ConfirmSplitRequest represents the request to confirm a split payment
type ConfirmSplitRequest struct {
	// No body needed - uses authenticated user (payer)
}

// DisputeSplitRequest represents the request to dispute a split
type DisputeSplitRequest struct {
	Reason string `json:"reason" validate:"required,min=1,max=500"`
}

// ExpenseResponse represents the response for an expense
type ExpenseResponse struct {
	ID            int64            `json:"id"`
	GroupID       int64            `json:"group_id"`
	PayerID       int64            `json:"payer_id"`
	PayerUsername string           `json:"payer_username,omitempty"`
	Description   string           `json:"description"`
	Amount        float64          `json:"amount"`
	ImageURL      *string          `json:"image_url,omitempty"`
	SplitType     string           `json:"split_type"`
	CreatedAt     string           `json:"created_at"`
	Splits        []*SplitResponse `json:"splits,omitempty"`
}

// SplitResponse represents the response for a split
type SplitResponse struct {
	ID               int64       `json:"id"`
	ExpenseID        int64       `json:"expense_id"`
	BorrowerID       int64       `json:"borrower_id"`
	BorrowerUsername string      `json:"borrower_username,omitempty"`
	AmountOwed       float64     `json:"amount_owed"`
	Status           SplitStatus `json:"status"`
	DisputeReason    *string     `json:"dispute_reason,omitempty"`
	SettlementID     *int64      `json:"settlement_id,omitempty"`
	UpdatedAt        string      `json:"updated_at"`
}

// ToResponse converts an Expense model to an ExpenseResponse DTO
func (e *Expense) ToResponse() *ExpenseResponse {
	return &ExpenseResponse{
		ID:            e.ID,
		GroupID:       e.GroupID,
		PayerID:       e.PayerID,
		PayerUsername: e.PayerUsername,
		Description:   e.Description,
		Amount:        e.Amount,
		ImageURL:      e.ImageURL,
		SplitType:     e.SplitType,
		CreatedAt:     e.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// ToResponse converts a Split model to a SplitResponse DTO
func (s *Split) ToResponse() *SplitResponse {
	return &SplitResponse{
		ID:               s.ID,
		ExpenseID:        s.ExpenseID,
		BorrowerID:       s.BorrowerID,
		BorrowerUsername: s.BorrowerUsername,
		AmountOwed:       s.AmountOwed,
		Status:           s.Status,
		DisputeReason:    s.DisputeReason,
		SettlementID:     s.SettlementID,
		UpdatedAt:        s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
