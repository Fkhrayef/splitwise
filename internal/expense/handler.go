package expense

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/fkhayef/splitwise/pkg/middleware"
	"github.com/fkhayef/splitwise/pkg/response"
)

// Handler handles HTTP requests for expense operations
type Handler struct {
	service *Service
}

// NewHandler creates a new expense handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the router for expense endpoints
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Delete("/{id}", h.Delete)

	r.Get("/group/{groupId}", h.ListByGroup)

	// Split operations
	r.Post("/splits/{splitId}/pay", h.MarkSplitAsPaid)
	r.Post("/splits/{splitId}/confirm", h.ConfirmSplitPayment)
	r.Post("/splits/{splitId}/dispute", h.DisputeSplit)

	return r
}

// Create handles POST /expenses
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	payerID, ok := middleware.GetUserID(r.Context())
	if !ok {
		payerID = 1
	}

	var req CreateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	validTypes := map[string]bool{"EVEN": true, "PERCENTAGE": true, "EXACT": true}
	if !validTypes[req.SplitType] {
		response.BadRequest(w, "Invalid split type. Must be EVEN, PERCENTAGE, or EXACT")
		return
	}

	result, err := h.service.CreateExpense(r.Context(), payerID, &req)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	expenseResp := result.Expense.ToResponse()
	expenseResp.Splits = make([]*SplitResponse, len(result.Splits))
	for i, s := range result.Splits {
		expenseResp.Splits[i] = s.ToResponse()
	}

	response.JSON(w, http.StatusCreated, expenseResp)
}

// GetByID handles GET /expenses/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid expense ID")
		return
	}

	result, err := h.service.GetExpenseByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrExpenseNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to get expense")
		return
	}

	expenseResp := result.Expense.ToResponse()
	expenseResp.Splits = make([]*SplitResponse, len(result.Splits))
	for i, s := range result.Splits {
		expenseResp.Splits[i] = s.ToResponse()
	}

	response.JSON(w, http.StatusOK, expenseResp)
}

// ListByGroup handles GET /expenses/group/{groupId}
func (h *Handler) ListByGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := strconv.ParseInt(chi.URLParam(r, "groupId"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid group ID")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	expenses, total, err := h.service.ListExpensesByGroupID(r.Context(), groupID, page, perPage)
	if err != nil {
		response.InternalError(w, "Failed to list expenses")
		return
	}

	expenseResponses := make([]*ExpenseResponse, len(expenses))
	for i, e := range expenses {
		expenseResponses[i] = e.ToResponse()
	}

	totalPages := (total + perPage - 1) / perPage
	meta := &response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}

	response.JSONWithMeta(w, http.StatusOK, expenseResponses, meta)
}

// Delete handles DELETE /expenses/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid expense ID")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	if err := h.service.DeleteExpense(r.Context(), id, userID); err != nil {
		if errors.Is(err, ErrExpenseNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		if errors.Is(err, ErrNotPayer) {
			response.Forbidden(w, err.Error())
			return
		}
		if errors.Is(err, ErrCannotDeleteExpense) {
			response.Conflict(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to delete expense")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Expense deleted successfully"})
}

// MarkSplitAsPaid handles POST /expenses/splits/{splitId}/pay
func (h *Handler) MarkSplitAsPaid(w http.ResponseWriter, r *http.Request) {
	splitID, err := strconv.ParseInt(chi.URLParam(r, "splitId"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid split ID")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	split, err := h.service.MarkSplitAsPaid(r.Context(), splitID, userID)
	if err != nil {
		if errors.Is(err, ErrSplitNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		if errors.Is(err, ErrNotBorrower) || errors.Is(err, ErrSplitLocked) || errors.Is(err, ErrInvalidStatusChange) {
			response.BadRequest(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to mark split as paid")
		return
	}

	response.JSON(w, http.StatusOK, split.ToResponse())
}

// ConfirmSplitPayment handles POST /expenses/splits/{splitId}/confirm
func (h *Handler) ConfirmSplitPayment(w http.ResponseWriter, r *http.Request) {
	splitID, err := strconv.ParseInt(chi.URLParam(r, "splitId"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid split ID")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	split, err := h.service.ConfirmSplitPayment(r.Context(), splitID, userID)
	if err != nil {
		if errors.Is(err, ErrSplitNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		if errors.Is(err, ErrNotPayer) || errors.Is(err, ErrSplitLocked) || errors.Is(err, ErrInvalidStatusChange) {
			response.BadRequest(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to confirm payment")
		return
	}

	response.JSON(w, http.StatusOK, split.ToResponse())
}

// DisputeSplit handles POST /expenses/splits/{splitId}/dispute
func (h *Handler) DisputeSplit(w http.ResponseWriter, r *http.Request) {
	splitID, err := strconv.ParseInt(chi.URLParam(r, "splitId"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid split ID")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1
	}

	var req DisputeSplitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if req.Reason == "" {
		response.BadRequest(w, "Dispute reason is required")
		return
	}

	split, err := h.service.DisputeSplit(r.Context(), splitID, userID, req.Reason)
	if err != nil {
		if errors.Is(err, ErrSplitNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		if errors.Is(err, ErrNotBorrower) || errors.Is(err, ErrInvalidStatusChange) {
			response.BadRequest(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to dispute split")
		return
	}

	response.JSON(w, http.StatusOK, split.ToResponse())
}
