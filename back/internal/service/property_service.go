package service

import (
	"context"
	"strings"

	"github.com/RoggerParedesTavara/mi-recibo/internal/domain"
	"github.com/RoggerParedesTavara/mi-recibo/internal/repository/postgres"
	"github.com/google/uuid"
)

type PropertyService struct {
	ownerRepo    *postgres.OwnerRepository
	propertyRepo *postgres.PropertyRepository
}

func NewPropertyService(ownerRepo *postgres.OwnerRepository, propertyRepo *postgres.PropertyRepository) *PropertyService {
	return &PropertyService{
		ownerRepo:    ownerRepo,
		propertyRepo: propertyRepo,
	}
}

func (s *PropertyService) CreateOwner(ctx context.Context, name, email, phone string) (*domain.Owner, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" || email == "" {
		return nil, domain.ErrValidation
	}

	owner := &domain.Owner{
		Name:  name,
		Email: email,
		Phone: phone,
	}
	if err := s.ownerRepo.Create(ctx, owner); err != nil {
		return nil, err
	}
	return owner, nil
}

func (s *PropertyService) GetOwner(ctx context.Context, id uuid.UUID) (*domain.Owner, error) {
	return s.ownerRepo.GetByID(ctx, id)
}

func (s *PropertyService) CreateProperty(ctx context.Context, ownerID uuid.UUID, name, address string) (*domain.Property, error) {
	name = strings.TrimSpace(name)
	address = strings.TrimSpace(address)
	if name == "" || address == "" {
		return nil, domain.ErrValidation
	}

	// Verify owner exists
	if _, err := s.ownerRepo.GetByID(ctx, ownerID); err != nil {
		return nil, err
	}

	property := &domain.Property{
		OwnerID: ownerID,
		Name:    name,
		Address: address,
	}
	if err := s.propertyRepo.Create(ctx, property); err != nil {
		return nil, err
	}
	return property, nil
}

func (s *PropertyService) GetProperty(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	return s.propertyRepo.GetByID(ctx, id)
}

func (s *PropertyService) ListPropertiesByOwner(ctx context.Context, ownerID uuid.UUID) ([]domain.Property, error) {
	return s.propertyRepo.ListByOwnerID(ctx, ownerID)
}
