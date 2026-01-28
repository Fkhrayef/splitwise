package group

import "time"

// MemberStatus represents the status of a group member
type MemberStatus string

const (
	MemberStatusInvited MemberStatus = "INVITED"
	MemberStatusJoined  MemberStatus = "JOINED"
)

// MemberRole represents the role of a group member
type MemberRole string

const (
	MemberRoleAdmin  MemberRole = "ADMIN"
	MemberRoleMember MemberRole = "MEMBER"
)

// Group represents a group in the system
type Group struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IsTemporary bool      `json:"is_temporary"`
	CreatedAt   time.Time `json:"created_at"`
}

// GroupMember represents a user's membership in a group
type GroupMember struct {
	ID       int64        `json:"id"`
	GroupID  int64        `json:"group_id"`
	UserID   int64        `json:"user_id"`
	Status   MemberStatus `json:"status"`
	Role     MemberRole   `json:"role"`
	JoinedAt time.Time    `json:"joined_at"`

	// Populated from JOIN
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
}
