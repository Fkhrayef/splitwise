package settlement

// CreateSettlementRequest represents the request to create a settlement
type CreateSettlementRequest struct {
	OtherUserID int64 `json:"other_user_id" validate:"required"` // The user you want to settle with
	// Payer/Receiver roles and Amount are calculated automatically based on net balance
}

// SettlementResponse represents the response for a settlement
type SettlementResponse struct {
	ID               int64            `json:"id"`
	PayerID          int64            `json:"payer_id"`
	PayerUsername    string           `json:"payer_username,omitempty"`
	ReceiverID       int64            `json:"receiver_id"`
	ReceiverUsername string           `json:"receiver_username,omitempty"`
	Amount           float64          `json:"amount"`
	CurrencyCode     string           `json:"currency_code"`
	Status           SettlementStatus `json:"status"`
	CreatedAt        string           `json:"created_at"`
}

// NetBalanceResponse represents the net balance with another user
type NetBalanceResponse struct {
	UserID   int64   `json:"user_id"`
	Username string  `json:"username"`
	Amount   float64 `json:"amount"`
	Message  string  `json:"message"` // e.g., "You owe John $50" or "John owes you $30"
}

// ToResponse converts a Settlement model to a SettlementResponse DTO
func (s *Settlement) ToResponse() *SettlementResponse {
	return &SettlementResponse{
		ID:               s.ID,
		PayerID:          s.PayerID,
		PayerUsername:    s.PayerUsername,
		ReceiverID:       s.ReceiverID,
		ReceiverUsername: s.ReceiverUsername,
		Amount:           s.Amount,
		CurrencyCode:     s.CurrencyCode,
		Status:           s.Status,
		CreatedAt:        s.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
