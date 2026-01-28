package user

import (
	"context"
	"errors"
)

// Common errors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailAlreadyInUse = errors.New("email already in use")
)

// Service handles user business logic
type Service struct {
	repo *Repository
}

// NewService creates a new user service with repository dependency injected
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new user
func (s *Service) Create(ctx context.Context, req *CreateUserRequest) (*User, error) {
	// Check if email is already in use
	existing, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyInUse
	}

	return s.repo.Create(ctx, req)
}

// GetByID retrieves a user by their ID
func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// List retrieves all users with pagination
func (s *Service) List(ctx context.Context, page, perPage int) ([]*User, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage
	return s.repo.List(ctx, perPage, offset)
}

// Update modifies an existing user
func (s *Service) Update(ctx context.Context, id int64, req *UpdateUserRequest) (*User, error) {
	// Check if user exists
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrUserNotFound
	}

	return s.repo.Update(ctx, id, req)
}

// Delete removes a user
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
