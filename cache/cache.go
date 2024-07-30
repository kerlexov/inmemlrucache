package cache

import (
	"fmt"
	"sync"
	"time"
)

type Node struct {
	Key   string
	Value any
	Prev  *Node
	Next  *Node
	TTL   *int64
	mu    sync.RWMutex
}

type Queue struct {
	Head *Node
	Tail *Node
	Len  int
	mu   sync.RWMutex
}

type Cache struct {
	Queue    Queue
	SMap     sync.Map
	Capacity int
}

func NewQueue() Queue {
	head := &Node{}
	tail := &Node{}
	head.Next = tail
	tail.Prev = head
	return Queue{Head: head, Tail: tail, mu: sync.RWMutex{}}
}

func NewCache(i int, cleanUp time.Duration) *Cache {
	k := &Cache{Queue: NewQueue(), SMap: sync.Map{}, Capacity: i}
	go k.StartCleanupTask(cleanUp)
	return k
}

func (k *Cache) Set(key string, value any, ttl ...time.Duration) {
	node := &Node{}
	var expiresAt *int64
	if ttl != nil {
		expTime := time.Now().Add(ttl[0]).Unix()
		expiresAt = &expTime
	}

	if existing, ok := k.SMap.Load(key); ok {
		node = k.Remove(existing.(*Node))
		node.Value = value
		node.TTL = expiresAt
	} else {
		node = &Node{Value: value, Key: key, TTL: expiresAt, mu: sync.RWMutex{}}
	}
	k.Add(node)
}

func (k *Cache) Remove(existing *Node) *Node {
	existing.mu.RLock()
	prev := existing.Prev
	next := existing.Next

	k.Queue.mu.Lock()
	prev.Next = next
	next.Prev = prev
	k.Queue.Len--
	k.Queue.mu.Unlock()

	k.SMap.Delete(existing.Key)
	defer existing.mu.RUnlock()
	return existing
}

func (k *Cache) Add(node *Node) *Node {
	k.Queue.mu.Lock()
	tmp := k.Queue.Head.Next
	node.mu.Lock()
	k.Queue.Head.Next = node
	node.Prev = k.Queue.Head
	node.Next = tmp
	tmp.Prev = node
	node.mu.Unlock()
	k.Queue.Len++
	k.Queue.mu.Unlock()

	if k.Queue.Len > k.Capacity {
		k.Remove(k.Queue.Tail.Prev)
	}

	k.SMap.Store(node.Key, node)
	return node
}

func (k *Cache) Get(key string) (any, bool) {
	tmp, ok := k.SMap.Load(key)
	if !ok || tmp == nil {
		return nil, false
	}

	node := tmp.(*Node)
	if node.TTL != nil && time.Now().Unix() > *node.TTL {
		k.Remove(node)
		return nil, false
	}

	k.Remove(node)
	k.Add(node)
	return node.Value, true
}

func (k *Cache) Display() {
	k.Queue.Display()
}

func (q *Queue) Display() {
	q.mu.RLock()
	defer q.mu.RUnlock()

	node := q.Head.Next
	fmt.Printf("Len: %d", q.Len)
	for i := 0; i < q.Len; i++ {
		fmt.Printf(" -> %v", node.Value)
		node = node.Next
	}
	fmt.Println()
}

func (k *Cache) RemoveExpired() {
	now := time.Now()
	k.Queue.mu.RLock()
	node := k.Queue.Head.Next
	k.Queue.mu.RUnlock()
	for i := 0; i < k.Queue.Len; i++ {
		next := node.Next
		if node.TTL != nil && now.Unix() > *node.TTL {
			k.Remove(node)
		}
		node = next
	}
}

func (k *Cache) StartCleanupTask(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		k.RemoveExpired()
	}
}

func lenSyncMap(m *sync.Map) int {
	var i int
	m.Range(func(k, v interface{}) bool {
		i++
		return true
	})
	return i
}

func (k *Cache) QueueLen() int {
	k.Queue.mu.RLock()
	tmpLen := k.Queue.Len
	k.Queue.mu.RUnlock()
	return tmpLen
}
