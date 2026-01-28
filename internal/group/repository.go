package group

import (
	"context"
	"database/sql"
	"fmt"
)

// Repository handles group data persistence
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new group repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new group into the database
func (r *Repository) Create(ctx context.Context, req *CreateGroupRequest) (*Group, error) {
	query := `
		INSERT INTO groups (name, description, is_temporary)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, is_temporary, created_at
	`

	group := &Group{}
	err := r.db.QueryRowContext(ctx, query, req.Name, req.Description, req.IsTemporary).Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.IsTemporary,
		&group.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	return group, nil
}

// GetByID retrieves a group by its ID
func (r *Repository) GetByID(ctx context.Context, id int64) (*Group, error) {
	query := `
		SELECT id, name, description, is_temporary, created_at
		FROM groups
		WHERE id = $1
	`

	group := &Group{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.IsTemporary,
		&group.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	return group, nil
}

// ListByUserID retrieves all groups for a user
func (r *Repository) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]*Group, int, error) {
	// Get total count
	var total int
	countQuery := `
		SELECT COUNT(DISTINCT g.id)
		FROM groups g
		JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = $1
	`
	if err := r.db.QueryRowContext(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count groups: %w", err)
	}

	// Get groups
	query := `
		SELECT g.id, g.name, g.description, g.is_temporary, g.created_at
		FROM groups g
		JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = $1
		ORDER BY g.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list groups: %w", err)
	}
	defer rows.Close()

	var groups []*Group
	for rows.Next() {
		group := &Group{}
		if err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.Description,
			&group.IsTemporary,
			&group.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, group)
	}

	return groups, total, nil
}

// Update modifies an existing group
func (r *Repository) Update(ctx context.Context, id int64, req *UpdateGroupRequest) (*Group, error) {
	query := `
		UPDATE groups
		SET name = COALESCE($2, name),
		    description = COALESCE($3, description)
		WHERE id = $1
		RETURNING id, name, description, is_temporary, created_at
	`

	group := &Group{}
	err := r.db.QueryRowContext(ctx, query, id, req.Name, req.Description).Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.IsTemporary,
		&group.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	return group, nil
}

// Delete removes a group from the database
func (r *Repository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM groups WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("group not found")
	}

	return nil
}

// AddMember adds a user to a group
func (r *Repository) AddMember(ctx context.Context, groupID int64, req *AddMemberRequest) (*GroupMember, error) {
	role := req.Role
	if role == "" {
		role = MemberRoleMember
	}

	query := `
		INSERT INTO group_members (group_id, user_id, status, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, group_id, user_id, status, role, joined_at
	`

	member := &GroupMember{}
	err := r.db.QueryRowContext(ctx, query, groupID, req.UserID, MemberStatusInvited, role).Scan(
		&member.ID,
		&member.GroupID,
		&member.UserID,
		&member.Status,
		&member.Role,
		&member.JoinedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	return member, nil
}

// GetMembers retrieves all members of a group
func (r *Repository) GetMembers(ctx context.Context, groupID int64) ([]*GroupMember, error) {
	query := `
		SELECT gm.id, gm.group_id, gm.user_id, gm.status, gm.role, gm.joined_at, u.username, u.email
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = $1
		ORDER BY gm.joined_at
	`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	defer rows.Close()

	var members []*GroupMember
	for rows.Next() {
		member := &GroupMember{}
		if err := rows.Scan(
			&member.ID,
			&member.GroupID,
			&member.UserID,
			&member.Status,
			&member.Role,
			&member.JoinedAt,
			&member.Username,
			&member.Email,
		); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, member)
	}

	return members, nil
}

// GetMember retrieves a specific member from a group
func (r *Repository) GetMember(ctx context.Context, groupID, userID int64) (*GroupMember, error) {
	query := `
		SELECT gm.id, gm.group_id, gm.user_id, gm.status, gm.role, gm.joined_at, u.username, u.email
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = $1 AND gm.user_id = $2
	`

	member := &GroupMember{}
	err := r.db.QueryRowContext(ctx, query, groupID, userID).Scan(
		&member.ID,
		&member.GroupID,
		&member.UserID,
		&member.Status,
		&member.Role,
		&member.JoinedAt,
		&member.Username,
		&member.Email,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	return member, nil
}

// UpdateMember updates a member's status or role
func (r *Repository) UpdateMember(ctx context.Context, groupID, userID int64, req *UpdateMemberRequest) (*GroupMember, error) {
	query := `
		UPDATE group_members
		SET status = COALESCE($3, status),
		    role = COALESCE($4, role)
		WHERE group_id = $1 AND user_id = $2
		RETURNING id, group_id, user_id, status, role, joined_at
	`

	member := &GroupMember{}
	err := r.db.QueryRowContext(ctx, query, groupID, userID, req.Status, req.Role).Scan(
		&member.ID,
		&member.GroupID,
		&member.UserID,
		&member.Status,
		&member.Role,
		&member.JoinedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to update member: %w", err)
	}

	return member, nil
}

// RemoveMember removes a user from a group
func (r *Repository) RemoveMember(ctx context.Context, groupID, userID int64) error {
	query := `DELETE FROM group_members WHERE group_id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("member not found")
	}

	return nil
}
