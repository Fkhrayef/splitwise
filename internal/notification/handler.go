package notification

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/fkhayef/splitwise/pkg/middleware"
	"github.com/fkhayef/splitwise/pkg/response"
)

// Handler handles HTTP requests for notification operations
type Handler struct {
	service *Service
}

// NewHandler creates a new notification handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the router for notification endpoints
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Get("/unread-count", h.GetUnreadCount)
	r.Post("/{id}/read", h.MarkAsRead)
	r.Post("/read-all", h.MarkAllAsRead)

	return r
}

// NotificationResponse represents the response for a notification
type NotificationResponse struct {
	ID                int64   `json:"id"`
	Message           string  `json:"message"`
	IsRead            bool    `json:"is_read"`
	RelatedEntityType *string `json:"related_entity_type,omitempty"`
	RelatedEntityID   *int64  `json:"related_entity_id,omitempty"`
	CreatedAt         string  `json:"created_at"`
}

// ToResponse converts a Notification to a NotificationResponse
func toResponse(n *Notification) *NotificationResponse {
	return &NotificationResponse{
		ID:                n.ID,
		Message:           n.Message,
		IsRead:            n.IsRead,
		RelatedEntityType: n.RelatedEntityType,
		RelatedEntityID:   n.RelatedEntityID,
		CreatedAt:         n.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// List handles GET /notifications
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	unreadOnly := r.URL.Query().Get("unread_only") == "true"

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	notifications, total, err := h.service.ListByRecipientID(r.Context(), userID, page, perPage, unreadOnly)
	if err != nil {
		response.InternalError(w, "Failed to list notifications")
		return
	}

	notificationResponses := make([]*NotificationResponse, len(notifications))
	for i, n := range notifications {
		notificationResponses[i] = toResponse(n)
	}

	totalPages := (total + perPage - 1) / perPage
	meta := &response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}

	response.JSONWithMeta(w, http.StatusOK, notificationResponses, meta)
}

// GetUnreadCount handles GET /notifications/unread-count
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	count, err := h.service.GetUnreadCount(r.Context(), userID)
	if err != nil {
		response.InternalError(w, "Failed to get unread count")
		return
	}

	response.JSON(w, http.StatusOK, map[string]int{"unread_count": count})
}

// MarkAsRead handles POST /notifications/{id}/read
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid notification ID")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	if err := h.service.MarkAsRead(r.Context(), id, userID); err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		if errors.Is(err, ErrNotRecipient) {
			response.Forbidden(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to mark notification as read")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Notification marked as read"})
}

// MarkAllAsRead handles POST /notifications/read-all
func (h *Handler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	if err := h.service.MarkAllAsRead(r.Context(), userID); err != nil {
		response.InternalError(w, "Failed to mark all notifications as read")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "All notifications marked as read"})
}
