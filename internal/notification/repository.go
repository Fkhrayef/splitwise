package notification

import (
	"context"
	"database/sql"
	"fmt"
)

// Repository handles notification data persistence
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new notification repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new notification into the database
func (r *Repository) Create(ctx context.Context, recipientID int64, message string, entityType *string, entityID *int64) (*Notification, error) {
	query := `
		INSERT INTO notifications (recipient_id, message, related_entity_type, related_entity_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, recipient_id, message, is_read, related_entity_type, related_entity_id, created_at
	`

	notification := &Notification{}
	err := r.db.QueryRowContext(ctx, query, recipientID, message, entityType, entityID).Scan(
		&notification.ID,
		&notification.RecipientID,
		&notification.Message,
		&notification.IsRead,
		&notification.RelatedEntityType,
		&notification.RelatedEntityID,
		&notification.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	return notification, nil
}

// GetByID retrieves a notification by its ID
func (r *Repository) GetByID(ctx context.Context, id int64) (*Notification, error) {
	query := `
		SELECT id, recipient_id, message, is_read, related_entity_type, related_entity_id, created_at
		FROM notifications
		WHERE id = $1
	`

	notification := &Notification{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&notification.ID,
		&notification.RecipientID,
		&notification.Message,
		&notification.IsRead,
		&notification.RelatedEntityType,
		&notification.RelatedEntityID,
		&notification.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return notification, nil
}

// ListByRecipientID retrieves all notifications for a user
func (r *Repository) ListByRecipientID(ctx context.Context, recipientID int64, limit, offset int, unreadOnly bool) ([]*Notification, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM notifications WHERE recipient_id = $1`
	if unreadOnly {
		countQuery += ` AND is_read = false`
	}
	if err := r.db.QueryRowContext(ctx, countQuery, recipientID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Get notifications
	query := `
		SELECT id, recipient_id, message, is_read, related_entity_type, related_entity_id, created_at
		FROM notifications
		WHERE recipient_id = $1
	`
	if unreadOnly {
		query += ` AND is_read = false`
	}
	query += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, recipientID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		notification := &Notification{}
		if err := rows.Scan(
			&notification.ID,
			&notification.RecipientID,
			&notification.Message,
			&notification.IsRead,
			&notification.RelatedEntityType,
			&notification.RelatedEntityID,
			&notification.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, notification)
	}

	return notifications, total, nil
}

// MarkAsRead marks a notification as read
func (r *Repository) MarkAsRead(ctx context.Context, id int64) error {
	query := `UPDATE notifications SET is_read = true WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}
	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (r *Repository) MarkAllAsRead(ctx context.Context, recipientID int64) error {
	query := `UPDATE notifications SET is_read = true WHERE recipient_id = $1 AND is_read = false`
	_, err := r.db.ExecContext(ctx, query, recipientID)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}
	return nil
}

// GetUnreadCount returns the count of unread notifications for a user
func (r *Repository) GetUnreadCount(ctx context.Context, recipientID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM notifications WHERE recipient_id = $1 AND is_read = false`
	if err := r.db.QueryRowContext(ctx, query, recipientID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count unread notifications: %w", err)
	}
	return count, nil
}
