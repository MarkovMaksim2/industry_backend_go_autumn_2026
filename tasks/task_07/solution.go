package main

import (
	"container/list"
	"sync"
)

type LRU[K comparable, V any] interface {
	Get(key K) (value V, ok bool)
	Set(key K, value V)
}
type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity int
	mu       sync.Mutex
	ll       list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element, 0),
	}
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.items[key]

	if !ok {
		var zero V
		return zero, ok
	}

	c.ll.MoveToBack(c.items[key])

	value = node.Value.(*entry[K, V]).value
	return value, ok
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		return
	}

	if node, ok := c.items[key]; ok {
		node.Value.(*entry[K, V]).value = value
		return
	}

	if len(c.items) >= c.capacity {
		keyToDelete := c.ll.Front().Value.(*entry[K, V]).key
		delete(c.items, keyToDelete)
		c.ll.Remove(c.ll.Front())
	}

	newNode := c.ll.PushBack(&entry[K, V]{key: key, value: value})

	c.items[key] = newNode
}
