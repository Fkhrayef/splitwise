package group

// CreateGroupRequest represents the request to create a new group
type CreateGroupRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Description *string `json:"description,omitempty"`
	IsTemporary bool    `json:"is_temporary"`
}

// UpdateGroupRequest represents the request to update a group
type UpdateGroupRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string `json:"description,omitempty"`
}

// AddMemberRequest represents the request to add a member to a group
type AddMemberRequest struct {
	UserID int64      `json:"user_id" validate:"required"`
	Role   MemberRole `json:"role"`
}

// UpdateMemberRequest represents the request to update a member's status or role
type UpdateMemberRequest struct {
	Status *MemberStatus `json:"status,omitempty"`
	Role   *MemberRole   `json:"role,omitempty"`
}

// GroupResponse represents the response for a group
type GroupResponse struct {
	ID          int64             `json:"id"`
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	IsTemporary bool              `json:"is_temporary"`
	CreatedAt   string            `json:"created_at"`
	Members     []*MemberResponse `json:"members,omitempty"`
}

// MemberResponse represents a member in a group response
type MemberResponse struct {
	ID       int64        `json:"id"`
	UserID   int64        `json:"user_id"`
	Username string       `json:"username"`
	Email    string       `json:"email"`
	Status   MemberStatus `json:"status"`
	Role     MemberRole   `json:"role"`
	JoinedAt string       `json:"joined_at"`
}

// ToResponse converts a Group model to a GroupResponse DTO
func (g *Group) ToResponse() *GroupResponse {
	return &GroupResponse{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		IsTemporary: g.IsTemporary,
		CreatedAt:   g.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// ToResponse converts a GroupMember model to a MemberResponse DTO
func (m *GroupMember) ToResponse() *MemberResponse {
	return &MemberResponse{
		ID:       m.ID,
		UserID:   m.UserID,
		Username: m.Username,
		Email:    m.Email,
		Status:   m.Status,
		Role:     m.Role,
		JoinedAt: m.JoinedAt.Format("2006-01-02T15:04:05Z"),
	}
}
