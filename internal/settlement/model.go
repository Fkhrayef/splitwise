package settlement

import "time"

// SettlementStatus represents the status of a settlement
type SettlementStatus string

const (
	SettlementStatusPending   SettlementStatus = "PENDING"
	SettlementStatusPaid      SettlementStatus = "PAID"
	SettlementStatusConfirmed SettlementStatus = "CONFIRMED"
	SettlementStatusRejected  SettlementStatus = "REJECTED"
)

// Settlement represents a bulk payment between two users
type Settlement struct {
	ID           int64            `json:"id"`
	PayerID      int64            `json:"payer_id"`      // Who sends the bulk money
	ReceiverID   int64            `json:"receiver_id"`   // Who receives the bulk money
	Amount       float64          `json:"amount"`        // The net amount
	CurrencyCode string           `json:"currency_code"`
	Status       SettlementStatus `json:"status"`
	CreatedAt    time.Time        `json:"created_at"`

	// Populated via JOIN
	PayerUsername    string `json:"payer_username,omitempty"`
	ReceiverUsername string `json:"receiver_username,omitempty"`
}

// NetBalance represents the net amount owed between two users
type NetBalance struct {
	UserID   int64   `json:"user_id"`
	Username string  `json:"username"`
	Amount   float64 `json:"amount"` // Positive = you owe them, Negative = they owe you
}
