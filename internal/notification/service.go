package notification

import (
	"context"
	"errors"
)

// Common errors
var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrNotRecipient         = errors.New("not the recipient of this notification")
)

// Service handles notification business logic
type Service struct {
	repo *Repository
}

// NewService creates a new notification service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new notification
func (s *Service) Create(ctx context.Context, recipientID int64, message string, entityType *string, entityID *int64) (*Notification, error) {
	return s.repo.Create(ctx, recipientID, message, entityType, entityID)
}

// GetByID retrieves a notification by its ID
func (s *Service) GetByID(ctx context.Context, id int64) (*Notification, error) {
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if notification == nil {
		return nil, ErrNotificationNotFound
	}
	return notification, nil
}

// ListByRecipientID retrieves all notifications for a user
func (s *Service) ListByRecipientID(ctx context.Context, recipientID int64, page, perPage int, unreadOnly bool) ([]*Notification, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage
	return s.repo.ListByRecipientID(ctx, recipientID, perPage, offset, unreadOnly)
}

// MarkAsRead marks a notification as read
func (s *Service) MarkAsRead(ctx context.Context, id, userID int64) error {
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if notification == nil {
		return ErrNotificationNotFound
	}
	if notification.RecipientID != userID {
		return ErrNotRecipient
	}

	return s.repo.MarkAsRead(ctx, id)
}

// MarkAllAsRead marks all notifications as read for a user
func (s *Service) MarkAllAsRead(ctx context.Context, userID int64) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}

// GetUnreadCount returns the count of unread notifications
func (s *Service) GetUnreadCount(ctx context.Context, userID int64) (int, error) {
	return s.repo.GetUnreadCount(ctx, userID)
}

// Helper methods for creating specific notification types

// NotifyGroupInvite creates a notification for a group invitation
func (s *Service) NotifyGroupInvite(ctx context.Context, recipientID int64, groupName string, groupID int64) (*Notification, error) {
	message := "You have been invited to join group: " + groupName
	entityType := "GROUP"
	return s.repo.Create(ctx, recipientID, message, &entityType, &groupID)
}

// NotifyExpenseAdded creates a notification for a new expense
func (s *Service) NotifyExpenseAdded(ctx context.Context, recipientID int64, payerName string, amount float64, expenseID int64) (*Notification, error) {
	message := payerName + " added an expense and you owe money"
	entityType := "EXPENSE"
	return s.repo.Create(ctx, recipientID, message, &entityType, &expenseID)
}

// NotifySplitPaid creates a notification when someone marks a split as paid
func (s *Service) NotifySplitPaid(ctx context.Context, recipientID int64, borrowerName string, splitID int64) (*Notification, error) {
	message := borrowerName + " says they paid you. Please confirm."
	entityType := "SPLIT"
	return s.repo.Create(ctx, recipientID, message, &entityType, &splitID)
}

// NotifySettlementCreated creates a notification for a new settlement
func (s *Service) NotifySettlementCreated(ctx context.Context, recipientID int64, payerName string, amount float64, settlementID int64) (*Notification, error) {
	message := payerName + " wants to settle up with you"
	entityType := "SETTLEMENT"
	return s.repo.Create(ctx, recipientID, message, &entityType, &settlementID)
}
