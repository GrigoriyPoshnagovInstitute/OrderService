package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/GrigoriyPoshnagovInstitute/OrderService/pkg/domain/model"
	"github.com/GrigoriyPoshnagovInstitute/OrderService/pkg/domain/service"
)

var _ model.OrderRepository = &mockOrderRepository{}

type mockOrderRepository struct {
	sync.RWMutex
	store map[uuid.UUID]*model.Order
}

func newMockOrderRepository() *mockOrderRepository {
	return &mockOrderRepository{
		store: make(map[uuid.UUID]*model.Order),
	}
}

func (m *mockOrderRepository) NextID() (uuid.UUID, error) {
	return uuid.NewV7()
}

func (m *mockOrderRepository) Store(order *model.Order) error {
	m.Lock()
	defer m.Unlock()
	m.store[order.ID] = order
	return nil
}

func (m *mockOrderRepository) Find(id uuid.UUID) (*model.Order, error) {
	m.RLock()
	defer m.RUnlock()
	order, ok := m.store[id]
	if !ok || order.DeletedAt != nil {
		return nil, model.ErrOrderNotFound
	}
	return order, nil
}

func (m *mockOrderRepository) Delete(id uuid.UUID) error {
	m.Lock()
	defer m.Unlock()
	order, ok := m.store[id]
	if !ok || order.DeletedAt != nil {
		return model.ErrOrderNotFound
	}
	now := time.Now().UTC()
	order.DeletedAt = &now
	return nil
}

var _ service.EventDispatcher = &mockEventDispatcher{}

type mockEventDispatcher struct {
	sync.Mutex
	events []service.Event
}

func (m *mockEventDispatcher) Dispatch(event service.Event) error {
	m.Lock()
	defer m.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *mockEventDispatcher) GetEvents() []service.Event {
	m.Lock()
	defer m.Unlock()
	evs := make([]service.Event, len(m.events))
	copy(evs, m.events)
	return evs
}

func (m *mockEventDispatcher) Clear() {
	m.Lock()
	defer m.Unlock()
	m.events = nil
}

func TestOrderService(t *testing.T) {
	setup := func(t *testing.T) (service.Order, *mockOrderRepository, *mockEventDispatcher) {
		repo := newMockOrderRepository()
		dispatcher := &mockEventDispatcher{}
		orderSvc := service.NewOrderService(repo, dispatcher)
		return orderSvc, repo, dispatcher
	}

	customerID := uuid.Must(uuid.NewV7())

	t.Run("should create an order successfully", func(t *testing.T) {
		orderSvc, repo, dispatcher := setup(t)

		orderID, err := orderSvc.CreateOrder(customerID)

		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, orderID)

		createdOrder, repoErr := repo.Find(orderID)
		require.NoError(t, repoErr)
		require.Equal(t, orderID, createdOrder.ID)
		require.Equal(t, customerID, createdOrder.CustomerID)
		require.Equal(t, model.Open, createdOrder.Status)

		events := dispatcher.GetEvents()
		require.Len(t, events, 1)
		createdEvent, ok := events[0].(model.OrderCreated)
		require.True(t, ok)
		require.Equal(t, orderID, createdEvent.OrderID)
		require.Equal(t, customerID, createdEvent.CustomerID)
	})

	t.Run("should add an item to an open order", func(t *testing.T) {
		orderSvc, repo, dispatcher := setup(t)
		orderID, _ := orderSvc.CreateOrder(customerID)
		dispatcher.Clear()

		productID := uuid.Must(uuid.NewV7())
		price := 150.50
		itemID, err := orderSvc.AddItem(orderID, productID, price)

		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, itemID)

		order, _ := repo.Find(orderID)
		require.Len(t, order.Items, 1)
		require.Equal(t, itemID, order.Items[0].ID)
		require.Equal(t, productID, order.Items[0].ProductID)
		require.Equal(t, price, order.Items[0].Price)

		events := dispatcher.GetEvents()
		require.Len(t, events, 1)
		itemsChangedEvent, ok := events[0].(model.OrderItemsChanged)
		require.True(t, ok)
		require.Equal(t, orderID, itemsChangedEvent.OrderID)
		require.Equal(t, []uuid.UUID{itemID}, itemsChangedEvent.AddedItems)
		require.Empty(t, itemsChangedEvent.RemovedItems)
	})

	t.Run("should fail to add item to a non-open order", func(t *testing.T) {
		orderSvc, repo, _ := setup(t)
		orderID, _ := orderSvc.CreateOrder(customerID)

		order, _ := repo.Find(orderID)
		order.Status = model.Paid
		repo.Store(order)

		_, err := orderSvc.AddItem(orderID, uuid.New(), 100)
		require.Error(t, err)
		require.Equal(t, service.ErrInvalidOrderStatus, err)
	})

	t.Run("should delete an item from an open order", func(t *testing.T) {
		orderSvc, repo, dispatcher := setup(t)
		orderID, _ := orderSvc.CreateOrder(customerID)
		productID := uuid.Must(uuid.NewV7())
		itemID, _ := orderSvc.AddItem(orderID, productID, 100)
		dispatcher.Clear()

		err := orderSvc.DeleteItem(orderID, itemID)
		require.NoError(t, err)

		order, _ := repo.Find(orderID)
		require.Empty(t, order.Items)

		events := dispatcher.GetEvents()
		require.Len(t, events, 1)
		itemsChangedEvent, ok := events[0].(model.OrderItemsChanged)
		require.True(t, ok)
		require.Equal(t, orderID, itemsChangedEvent.OrderID)
		require.Equal(t, []uuid.UUID{itemID}, itemsChangedEvent.RemovedItems)
		require.Empty(t, itemsChangedEvent.AddedItems)
	})

	t.Run("should fail to delete a non-existent item", func(t *testing.T) {
		orderSvc, _, _ := setup(t)
		orderID, _ := orderSvc.CreateOrder(customerID)

		err := orderSvc.DeleteItem(orderID, uuid.Must(uuid.NewV7()))
		require.ErrorIs(t, err, service.ErrItemNotFound)
	})

	t.Run("should set a new status for an order", func(t *testing.T) {
		orderSvc, repo, dispatcher := setup(t)
		orderID, _ := orderSvc.CreateOrder(customerID)
		dispatcher.Clear()

		err := orderSvc.SetStatus(orderID, model.Paid)
		require.NoError(t, err)

		order, _ := repo.Find(orderID)
		require.Equal(t, model.Paid, order.Status)

		events := dispatcher.GetEvents()
		require.Len(t, events, 1)
		statusChangedEvent, ok := events[0].(model.OrderStatusChanged)
		require.True(t, ok)
		require.Equal(t, orderID, statusChangedEvent.OrderID)
		require.Equal(t, model.Paid, statusChangedEvent.NewStatus)
	})

	t.Run("should soft delete an order", func(t *testing.T) {
		orderSvc, repo, dispatcher := setup(t)
		orderID, _ := orderSvc.CreateOrder(customerID)
		dispatcher.Clear()

		err := orderSvc.DeleteOrder(orderID)
		require.NoError(t, err)

		_, findErr := repo.Find(orderID)
		require.ErrorIs(t, findErr, model.ErrOrderNotFound)

		repo.RLock()
		deletedOrder := repo.store[orderID]
		repo.RUnlock()
		require.NotNil(t, deletedOrder.DeletedAt)

		events := dispatcher.GetEvents()
		require.Len(t, events, 1)
		deletedEvent, ok := events[0].(model.OrderDeleted)
		require.True(t, ok)
		require.Equal(t, orderID, deletedEvent.OrderID)
	})
}
