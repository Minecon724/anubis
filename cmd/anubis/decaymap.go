package main

import (
	"sync"
	"time"
)

func zilch[T any]() T {
	var zero T
	return zero
}

// DecayMap is a lazy key->value map. It's a wrapper around a map and a mutex. If values exceed their time-to-live, they are pruned at Get time.
type DecayMap[K comparable, V any] struct {
	data map[K]decayMapEntry[V]
	lock sync.RWMutex
}

type decayMapEntry[V any] struct {
	Value  V
	expiry time.Time
}

// NewDecayMap creates a new DecayMap of key type K and value type V.
//
// Key types must be comparable to work with maps.
func NewDecayMap[K comparable, V any]() *DecayMap[K, V] {
	return &DecayMap[K, V]{
		data: make(map[K]decayMapEntry[V]),
	}
}

// expire forcibly expires a key by setting its time-to-live one second in the past.
func (m *DecayMap[K, V]) expire(key K) bool {
	m.lock.RLock()
	val, ok := m.data[key]
	m.lock.RUnlock()

	if !ok {
		return false
	}

	m.lock.Lock()
	val.expiry = time.Now().Add(-1 * time.Second)
	m.data[key] = val
	m.lock.Unlock()

	return true
}

// Get gets a value from the DecayMap by key.
//
// If a value has expired, forcibly delete it if it was not updated.
func (m *DecayMap[K, V]) Get(key K) (V, bool) {
	m.lock.RLock()
	value, ok := m.data[key]
	m.lock.RUnlock()

	if !ok {
		return zilch[V](), false
	}

	if time.Now().After(value.expiry) {
		m.lock.Lock()
		// Since previously reading m.data[key], the value may have been updated.
		// Delete the entry only if the expiry time is still the same.
		if m.data[key].expiry == value.expiry {
			delete(m.data, key)
		}
		m.lock.Unlock()

		return zilch[V](), false
	}

	return value.Value, true
}

// Set sets a key value pair in the map.
func (m *DecayMap[K, V]) Set(key K, value V, ttl time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.data[key] = decayMapEntry[V]{
		Value:  value,
		expiry: time.Now().Add(ttl),
	}
}
