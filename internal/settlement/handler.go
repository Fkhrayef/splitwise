package settlement

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/fkhayef/splitwise/pkg/middleware"
	"github.com/fkhayef/splitwise/pkg/response"
)

// Handler handles HTTP requests for settlement operations
type Handler struct {
	service *Service
}

// NewHandler creates a new settlement handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the router for settlement endpoints
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/balances", h.GetNetBalances)
	r.Get("/balances/{userId}", h.GetNetBalanceWithUser)
	r.Get("/{id}", h.GetByID)
	r.Post("/{id}/pay", h.MarkAsPaid)
	r.Post("/{id}/confirm", h.Confirm)
	r.Post("/{id}/reject", h.Reject)

	return r
}

// Create handles POST /settlements
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	payerID, ok := middleware.GetUserID(r.Context())
	if !ok {
		payerID = 1
	}

	var req CreateSettlementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	settlement, err := h.service.CreateSettlement(r.Context(), payerID, &req)
	if err != nil {
		if errors.Is(err, ErrAlreadySettled) {
			response.BadRequest(w, err.Error())
			return
		}
		if errors.Is(err, ErrCannotSettleSelf) {
			response.BadRequest(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to create settlement")
		return
	}

	response.JSON(w, http.StatusCreated, settlement.ToResponse())
}

// GetByID handles GET /settlements/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid settlement ID")
		return
	}

	settlement, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrSettlementNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to get settlement")
		return
	}

	response.JSON(w, http.StatusOK, settlement.ToResponse())
}

// List handles GET /settlements
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	settlements, total, err := h.service.ListByUserID(r.Context(), userID, page, perPage)
	if err != nil {
		response.InternalError(w, "Failed to list settlements")
		return
	}

	settlementResponses := make([]*SettlementResponse, len(settlements))
	for i, s := range settlements {
		settlementResponses[i] = s.ToResponse()
	}

	totalPages := (total + perPage - 1) / perPage
	meta := &response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}

	response.JSONWithMeta(w, http.StatusOK, settlementResponses, meta)
}

// MarkAsPaid handles POST /settlements/{id}/pay
func (h *Handler) MarkAsPaid(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid settlement ID")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	settlement, err := h.service.MarkAsPaid(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, ErrSettlementNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		if errors.Is(err, ErrNotPayer) || errors.Is(err, ErrInvalidStatusChange) {
			response.BadRequest(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to mark settlement as paid")
		return
	}

	response.JSON(w, http.StatusOK, settlement.ToResponse())
}

// Confirm handles POST /settlements/{id}/confirm
func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid settlement ID")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	settlement, err := h.service.Confirm(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, ErrSettlementNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		if errors.Is(err, ErrNotReceiver) || errors.Is(err, ErrInvalidStatusChange) {
			response.BadRequest(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to confirm settlement")
		return
	}

	response.JSON(w, http.StatusOK, settlement.ToResponse())
}

// Reject handles POST /settlements/{id}/reject
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid settlement ID")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	settlement, err := h.service.Reject(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, ErrSettlementNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		if errors.Is(err, ErrNotReceiver) || errors.Is(err, ErrInvalidStatusChange) {
			response.BadRequest(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to reject settlement")
		return
	}

	response.JSON(w, http.StatusOK, settlement.ToResponse())
}

// GetNetBalances handles GET /settlements/balances
func (h *Handler) GetNetBalances(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	balances, err := h.service.GetNetBalances(r.Context(), userID)
	if err != nil {
		response.InternalError(w, "Failed to get net balances")
		return
	}

	response.JSON(w, http.StatusOK, balances)
}

// GetNetBalanceWithUser handles GET /settlements/balances/{userId}
func (h *Handler) GetNetBalanceWithUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	otherUserID, err := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	balance, err := h.service.GetNetBalanceWithUser(r.Context(), userID, otherUserID, "User")
	if err != nil {
		response.InternalError(w, "Failed to get net balance")
		return
	}

	response.JSON(w, http.StatusOK, balance)
}
