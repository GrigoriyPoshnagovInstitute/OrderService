package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrOrderNotFound = errors.New("order not found")

type OrderStatus int

const (
	Open OrderStatus = iota
	Pending
	Paid
	Cancelled
)

type Order struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	Status     OrderStatus
	Items      []Item
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

type Item struct {
	ID        uuid.UUID
	ProductID uuid.UUID
	Price     float64
}

type OrderRepository interface {
	NextID() (uuid.UUID, error)
	Store(order *Order) error
	Find(id uuid.UUID) (*Order, error)
	Delete(id uuid.UUID) error
}
