package bolt_db

import "sync"

type InMemKV[K comparable, V any] struct {
	lck   sync.RWMutex
	items map[K]V
}

func (m *InMemKV[K, V]) Get(k K) V {
	m.lck.RLock()
	defer m.lck.RUnlock()
	return m.items[k]
}

func (m *InMemKV[K, V]) Set(k K, v V) {
	m.lck.Lock()
	defer m.lck.Unlock()
	m.items[k] = v
}

func (m *InMemKV[K, V]) Delete(k K) {
	m.lck.Lock()
	defer m.lck.Unlock()
	delete(m.items, k)
}

func (m *InMemKV[K, V]) Exists(k K) bool {
	m.lck.RLock()
	defer m.lck.RUnlock()
	_, ok := m.items[k]
	return ok
}

func (m *InMemKV[K, V]) Len() int {
	m.lck.RLock()
	defer m.lck.RUnlock()
	return len(m.items)
}

func (m *InMemKV[K, V]) Clear() {
	m.lck.Lock()
	defer m.lck.Unlock()
	m.items = make(map[K]V, 10_000)
}

func (m *InMemKV[K, V]) ForEach(fn func(k K, v V) error) error {
	m.lck.RLock()
	defer m.lck.RUnlock()
	for k, v := range m.items {
		err := fn(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewInmemMap[K comparable, V any]() *InMemKV[K, V] {
	return &InMemKV[K, V]{
		items: make(map[K]V, 10_000),
	}
}
