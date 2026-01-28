package group

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/fkhayef/splitwise/pkg/middleware"
	"github.com/fkhayef/splitwise/pkg/response"
)

// Handler handles HTTP requests for group operations
type Handler struct {
	service *Service
}

// NewHandler creates a new group handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the router for group endpoints
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	// Member management
	r.Post("/{id}/members", h.AddMember)
	r.Get("/{id}/members", h.GetMembers)
	r.Put("/{id}/members/{userId}", h.UpdateMember)
	r.Delete("/{id}/members/{userId}", h.RemoveMember)
	r.Post("/{id}/accept", h.AcceptInvitation)

	return r
}

// Create handles POST /groups
// @Summary      Create a new group
// @Description  Create a new group and add creator as admin
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        request body CreateGroupRequest true "Group creation request"
// @Success      201 {object} response.APIResponse{data=GroupResponse}
// @Failure      400 {object} response.APIResponse
// @Router       /groups [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	// Get creator ID from context (set by auth middleware)
	creatorID, ok := middleware.GetUserID(r.Context())
	if !ok {
		// For now, use a default user ID if not authenticated
		creatorID = 1
	}

	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	group, err := h.service.Create(r.Context(), creatorID, &req)
	if err != nil {
		response.InternalError(w, "Failed to create group")
		return
	}

	response.JSON(w, http.StatusCreated, group.ToResponse())
}

// GetByID handles GET /groups/{id}
// @Summary      Get group by ID
// @Description  Get a group with all its members
// @Tags         groups
// @Produce      json
// @Param        id path int true "Group ID"
// @Success      200 {object} response.APIResponse{data=GroupResponse}
// @Failure      404 {object} response.APIResponse
// @Router       /groups/{id} [get]
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid group ID")
		return
	}

	group, members, err := h.service.GetByIDWithMembers(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to get group")
		return
	}

	groupResp := group.ToResponse()
	groupResp.Members = make([]*MemberResponse, len(members))
	for i, m := range members {
		groupResp.Members[i] = m.ToResponse()
	}

	response.JSON(w, http.StatusOK, groupResp)
}

// List handles GET /groups
// @Summary      List my groups
// @Description  Get a paginated list of groups for the current user
// @Tags         groups
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        per_page query int false "Items per page" default(20)
// @Success      200 {object} response.APIResponse{data=[]GroupResponse}
// @Router       /groups [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1 // Default for development
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	groups, total, err := h.service.ListByUserID(r.Context(), userID, page, perPage)
	if err != nil {
		response.InternalError(w, "Failed to list groups")
		return
	}

	groupResponses := make([]*GroupResponse, len(groups))
	for i, group := range groups {
		groupResponses[i] = group.ToResponse()
	}

	totalPages := (total + perPage - 1) / perPage
	meta := &response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}

	response.JSONWithMeta(w, http.StatusOK, groupResponses, meta)
}

// Update handles PUT /groups/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid group ID")
		return
	}

	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	group, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to update group")
		return
	}

	response.JSON(w, http.StatusOK, group.ToResponse())
}

// Delete handles DELETE /groups/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid group ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		response.InternalError(w, "Failed to delete group")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Group deleted successfully"})
}

// AddMember handles POST /groups/{id}/members
// @Summary      Add member to group
// @Description  Invite a user to join the group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        id path int true "Group ID"
// @Param        request body AddMemberRequest true "Member to add"
// @Success      201 {object} response.APIResponse{data=MemberResponse}
// @Failure      404 {object} response.APIResponse
// @Failure      409 {object} response.APIResponse
// @Router       /groups/{id}/members [post]
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid group ID")
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	member, err := h.service.AddMember(r.Context(), groupID, &req)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		if errors.Is(err, ErrMemberAlreadyExists) {
			response.Conflict(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to add member")
		return
	}

	response.JSON(w, http.StatusCreated, member.ToResponse())
}

// GetMembers handles GET /groups/{id}/members
func (h *Handler) GetMembers(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	groupID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid group ID")
		return
	}

	members, err := h.service.GetMembers(r.Context(), groupID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to get members")
		return
	}

	memberResponses := make([]*MemberResponse, len(members))
	for i, m := range members {
		memberResponses[i] = m.ToResponse()
	}

	response.JSON(w, http.StatusOK, memberResponses)
}

// UpdateMember handles PUT /groups/{id}/members/{userId}
func (h *Handler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid group ID")
		return
	}

	userID, err := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	var req UpdateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	member, err := h.service.UpdateMember(r.Context(), groupID, userID, &req)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to update member")
		return
	}

	response.JSON(w, http.StatusOK, member.ToResponse())
}

// RemoveMember handles DELETE /groups/{id}/members/{userId}
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid group ID")
		return
	}

	userID, err := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	if err := h.service.RemoveMember(r.Context(), groupID, userID); err != nil {
		response.InternalError(w, "Failed to remove member")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Member removed successfully"})
}

// AcceptInvitation handles POST /groups/{id}/accept
func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid group ID")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		userID = 1 // Default for development
	}

	member, err := h.service.AcceptInvitation(r.Context(), groupID, userID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			response.NotFound(w, "You are not invited to this group")
			return
		}
		response.InternalError(w, "Failed to accept invitation")
		return
	}

	response.JSON(w, http.StatusOK, member.ToResponse())
}
