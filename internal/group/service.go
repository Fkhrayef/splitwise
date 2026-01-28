package group

import (
	"context"
	"errors"
)

// Common errors
var (
	ErrGroupNotFound       = errors.New("group not found")
	ErrMemberNotFound      = errors.New("member not found")
	ErrMemberAlreadyExists = errors.New("user is already a member of this group")
	ErrNotAuthorized       = errors.New("not authorized to perform this action")
)

// Service handles group business logic
type Service struct {
	repo *Repository
}

// NewService creates a new group service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new group and adds the creator as admin
func (s *Service) Create(ctx context.Context, creatorID int64, req *CreateGroupRequest) (*Group, error) {
	// Create the group
	group, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add creator as admin
	_, err = s.repo.AddMember(ctx, group.ID, &AddMemberRequest{
		UserID: creatorID,
		Role:   MemberRoleAdmin,
	})
	if err != nil {
		// TODO: Should rollback group creation in a transaction
		return nil, err
	}

	// Update the admin's status to JOINED immediately
	_, err = s.repo.UpdateMember(ctx, group.ID, creatorID, &UpdateMemberRequest{
		Status: statusPtr(MemberStatusJoined),
	})
	if err != nil {
		return nil, err
	}

	return group, nil
}

// GetByID retrieves a group by its ID
func (s *Service) GetByID(ctx context.Context, id int64) (*Group, error) {
	group, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}
	return group, nil
}

// GetByIDWithMembers retrieves a group with all its members
func (s *Service) GetByIDWithMembers(ctx context.Context, id int64) (*Group, []*GroupMember, error) {
	group, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	members, err := s.repo.GetMembers(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return group, members, nil
}

// ListByUserID retrieves all groups for a user
func (s *Service) ListByUserID(ctx context.Context, userID int64, page, perPage int) ([]*Group, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage
	return s.repo.ListByUserID(ctx, userID, perPage, offset)
}

// Update modifies an existing group
func (s *Service) Update(ctx context.Context, id int64, req *UpdateGroupRequest) (*Group, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrGroupNotFound
	}

	return s.repo.Update(ctx, id, req)
}

// Delete removes a group
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// AddMember adds a user to a group
func (s *Service) AddMember(ctx context.Context, groupID int64, req *AddMemberRequest) (*GroupMember, error) {
	// Check if group exists
	group, err := s.repo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}

	// Check if user is already a member
	existing, err := s.repo.GetMember(ctx, groupID, req.UserID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrMemberAlreadyExists
	}

	return s.repo.AddMember(ctx, groupID, req)
}

// GetMembers retrieves all members of a group
func (s *Service) GetMembers(ctx context.Context, groupID int64) ([]*GroupMember, error) {
	// Check if group exists
	group, err := s.repo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}

	return s.repo.GetMembers(ctx, groupID)
}

// UpdateMember updates a member's status or role
func (s *Service) UpdateMember(ctx context.Context, groupID, userID int64, req *UpdateMemberRequest) (*GroupMember, error) {
	member, err := s.repo.UpdateMember(ctx, groupID, userID, req)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrMemberNotFound
	}
	return member, nil
}

// RemoveMember removes a user from a group
func (s *Service) RemoveMember(ctx context.Context, groupID, userID int64) error {
	return s.repo.RemoveMember(ctx, groupID, userID)
}

// AcceptInvitation allows a user to accept their group invitation
func (s *Service) AcceptInvitation(ctx context.Context, groupID, userID int64) (*GroupMember, error) {
	// Check if user is a member with INVITED status
	member, err := s.repo.GetMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrMemberNotFound
	}
	if member.Status != MemberStatusInvited {
		return member, nil // Already joined
	}

	return s.repo.UpdateMember(ctx, groupID, userID, &UpdateMemberRequest{
		Status: statusPtr(MemberStatusJoined),
	})
}

// Helper function to get a pointer to a MemberStatus
func statusPtr(s MemberStatus) *MemberStatus {
	return &s
}
