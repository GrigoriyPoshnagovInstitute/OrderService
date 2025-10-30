package model

import "github.com/google/uuid"

type OrderCreated struct {
	OrderID    uuid.UUID
	CustomerID uuid.UUID
}

func (e OrderCreated) Type() string {
	return "OrderCreated"
}

type OrderItemsChanged struct {
	OrderID      uuid.UUID
	AddedItems   []uuid.UUID
	RemovedItems []uuid.UUID
}

func (e OrderItemsChanged) Type() string {
	return "OrderItemsChanged"
}

type OrderStatusChanged struct {
	OrderID   uuid.UUID
	NewStatus OrderStatus
}

func (e OrderStatusChanged) Type() string {
	return "OrderStatusChanged"
}

type OrderDeleted struct {
	OrderID uuid.UUID
}

func (e OrderDeleted) Type() string {
	return "OrderDeleted"
}
