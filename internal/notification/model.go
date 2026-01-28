package notification

import "time"

// Notification represents a notification in the system
type Notification struct {
	ID                int64     `json:"id"`
	RecipientID       int64     `json:"recipient_id"`
	Message           string    `json:"message"`
	IsRead            bool      `json:"is_read"`
	RelatedEntityType *string   `json:"related_entity_type,omitempty"` // e.g., "SETTLEMENT", "EXPENSE", "GROUP"
	RelatedEntityID   *int64    `json:"related_entity_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeGroupInvite    NotificationType = "GROUP_INVITE"
	NotificationTypeExpenseAdded   NotificationType = "EXPENSE_ADDED"
	NotificationTypeSplitAssigned  NotificationType = "SPLIT_ASSIGNED"
	NotificationTypeSplitPaid      NotificationType = "SPLIT_PAID"
	NotificationTypeSplitConfirmed NotificationType = "SPLIT_CONFIRMED"
	NotificationTypeSettlement     NotificationType = "SETTLEMENT"
)
