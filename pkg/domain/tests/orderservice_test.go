package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"order/pkg/domain/model"
	"order/pkg/domain/service"
)

func TestOrderService(t *testing.T) {
	repo := &mockOrderRepository{
		store: map[uuid.UUID]*model.Order{},
	}
	eventDispatcher := &mockEventDispatcher{}

	orderService := service.NewOrderService(repo, eventDispatcher)

	customerID := uuid.Must(uuid.NewV7())
	t.Run("Create order", func(t *testing.T) {
		orderID, err := orderService.CreateOrder(customerID)
		require.NoError(t, err)

		require.NotNil(t, repo.store[orderID])
		require.Len(t, eventDispatcher.events, 1)
		require.Equal(t, model.OrderCreated{}.Type(), eventDispatcher.events[0].Type())
	})
}

var _ model.OrderRepository = &mockOrderRepository{}

type mockOrderRepository struct {
	store map[uuid.UUID]*model.Order
}

func (m mockOrderRepository) NextID() (uuid.UUID, error) {
	return uuid.NewV7()
}

func (m mockOrderRepository) Store(order *model.Order) error {
	m.store[order.ID] = order
	return nil
}

func (m mockOrderRepository) Find(id uuid.UUID) (*model.Order, error) {
	if order, ok := m.store[id]; ok && order.DeletedAt != nil {
		return order, nil
	}
	return nil, model.ErrOrderNotFound
}

func (m mockOrderRepository) Delete(id uuid.UUID) error {
	if order, ok := m.store[id]; ok && order.DeletedAt != nil {
		order.DeletedAt = toPtr(time.Now())
		return nil
	}
	return model.ErrOrderNotFound
}

var _ service.EventDispatcher = &mockEventDispatcher{}

type mockEventDispatcher struct {
	events []service.Event
}

func (m mockEventDispatcher) Dispatch(event service.Event) error {
	m.events = append(m.events, event)
	return nil
}

func toPtr[V any](v V) *V {
	return &v
}
