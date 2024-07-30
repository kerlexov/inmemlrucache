package cache

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	k := NewCache(3, time.Second*5)
	if k.Capacity != 3 {
		t.Errorf("Expected capacity 3, got %d", k.Capacity)
	}
	if k.Queue.Len != 0 {
		t.Errorf("Expected queue length 0, got %d", k.Queue.Len)
	}
	if lenSyncMap(&k.SMap) != 0 {
		t.Errorf("Expected hash length 0, got %d", lenSyncMap(&k.SMap))
	}
}

func TestCacheSetAndGet(t *testing.T) {
	k := NewCache(3, time.Second*5)

	k.Set("key1", "value1")
	v, ok := k.Get("key1")
	if !ok || v != "value1" {
		t.Errorf("Expected value1, got %v", v)
	}

	k.Set("key2", "value2")
	k.Set("key3", "value3")

	// Test overwriting existing key
	k.Set("key2", "new_value2")
	v, ok = k.Get("key2")
	if !ok || v != "new_value2" {
		t.Errorf("Expected new_value2, got %v", v)
	}

	// Test capacity limit
	k.Set("key4", "value4")
	_, ok = k.Get("key1")
	if ok {
		t.Error("Expected key1 to be removed due to capacity limit")
	}

	if k.Queue.Len != 3 {
		t.Errorf("Expected queue length 3, got %d", k.Queue.Len)
	}
}

func TestCacheRemove(t *testing.T) {
	k := NewCache(3, time.Second*5)

	k.Set("key1", "value1")
	k.Set("key2", "value2")

	tmp, ok := k.SMap.Load("key1")
	if !ok {
		t.Fatal("Expected key1 to exist in hash")
	}
	node := tmp.(*Node)
	k.Remove(node)

	_, ok = k.Get("key1")
	if ok {
		t.Error("Expected key1 to be removed")
	}

	if k.Queue.Len != 1 {
		t.Errorf("Expected queue length 1, got %d", k.Queue.Len)
	}
}

func TestCacheOrder(t *testing.T) {
	k := NewCache(3, time.Second*5)

	k.Set("key1", "value1")
	k.Set("key2", "value2")
	k.Set("key3", "value3")

	// Access key2 to move it to the front
	k.Get("key2")

	node := k.Queue.Head.Next
	expectedOrder := []string{"key2", "key3", "key1"}

	for i, expected := range expectedOrder {
		if node.Key != expected {
			t.Errorf("Expected %s at position %d, got %s", expected, i, node.Key)
		}
		node = node.Next
	}
}

func TestCacheTTL(t *testing.T) {
	k := NewCache(3, time.Second*5)

	// Test setting with TTL
	k.Set("key1", "value1", 5*time.Second)
	time.Sleep(1 * time.Second)
	v, ok := k.Get("key1")
	if !ok || v != "value1" {
		t.Errorf("Expected value1, got %v", v)
	}

	// Wait for expiration
	time.Sleep(5 * time.Second)

	// Try to get expired key
	v, ok = k.Get("key1")
	if ok {
		t.Errorf("Expected key1 to be expired, but got %v", v)
	}

	// Test setting without TTL
	k.Set("key2", "value2")
	time.Sleep(3 * time.Second)
	v, ok = k.Get("key2")
	if !ok || v != "value2" {
		t.Errorf("Expected value2, got %v", v)
	}

	// Test updating TTL
	k.Set("key3", "value3", 2*time.Second)
	time.Sleep(1 * time.Second)
	k.Set("key3", "value3_updated", 20*time.Second)
	time.Sleep(2 * time.Second)
	v, ok = k.Get("key3")
	if !ok || v != "value3_updated" {
		t.Errorf("Expected value3_updated, got %v", v)
	}

	// Test removing expired items doesn't affect unexpired items
	k.Set("key4", "value4", 1*time.Second)
	k.Set("key5", "value5", 15*time.Second)
	time.Sleep(2 * time.Second)
	k.RemoveExpired()
	_, ok = k.Get("key4")
	if ok {
		t.Errorf("Expected key4 to be removed")
	}
	v, ok = k.Get("key5")
	if !ok || v != "value5" {
		t.Errorf("Expected value5, got %v", v)
	}
}

func TestCacheCapacityWithTTL(t *testing.T) {
	k := NewCache(2, time.Second*5)

	k.Set("key1", "value1", 10*time.Second)
	k.Set("key2", "value2", 60*time.Second)
	k.Set("key3", "value3", 1*time.Second)

	// Check that the oldest item (key1) was removed due to capacity
	_, ok := k.Get("key1")
	if ok {
		t.Error("Expected key1 to be removed due to capacity limit")
	}

	// Wait for key3 to expire
	time.Sleep(7 * time.Second)

	// This should not panic, as key3 should be removed due to expiration
	k.Set("key4", "value4")

	v, ok := k.Get("key2")
	if !ok || v != "value2" {
		t.Errorf("Expected value2, got %v", v)
	}

	v, ok = k.Get("key4")
	if !ok || v != "value4" {
		t.Errorf("Expected value4, got %v", v)
	}
}

func TestCacheRemoveExpired(t *testing.T) {
	k := NewCache(5, time.Second*5)

	k.Set("key1", "value1", 1*time.Second)
	k.Set("key2", "value2", 2*time.Second)
	k.Set("key3", "value3", 12*time.Second)
	k.Set("key4", "value4") // No TTL

	time.Sleep(7 * time.Second)

	k.RemoveExpired()

	_, ok := k.Get("key1")
	if ok {
		t.Error("Expected key1 to be removed")
	}

	_, ok = k.Get("key2")
	if ok {
		t.Error("Expected key2 to be removed")
	}

	v, ok := k.Get("key3")
	if !ok || v != "value3" {
		t.Errorf("Expected value3, got %v", v)
	}

	v, ok = k.Get("key4")
	if !ok || v != "value4" {
		t.Errorf("Expected value4, got %v", v)
	}
}

func TestCacheTTLOverwrite(t *testing.T) {
	k := NewCache(3, time.Second*5)

	// Set initial value with TTL
	k.Set("key1", "value1", 2*time.Second)

	// Overwrite with new value and longer TTL
	k.Set("key1", "value1_updated", 4*time.Second)

	time.Sleep(3 * time.Second)

	// Check that the key still exists after the original TTL would have expired
	v, ok := k.Get("key1")
	if !ok || v != "value1_updated" {
		t.Errorf("Expected value1_updated, got %v", v)
	}

	// Wait for the new TTL to expire
	time.Sleep(2 * time.Second)

	// Check that the key has now expired
	_, ok = k.Get("key1")
	if ok {
		t.Error("Expected key1 to be expired")
	}
}

func TestConcurrentAccess(t *testing.T) {
	k := NewCache(1000, time.Second*5)
	var wg sync.WaitGroup
	numOps := 10
	numGoroutines := 2

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := fmt.Sprintf("key%d-%d", id, j)
				value := fmt.Sprintf("value%d-%d", id, j)
				k.Set(key, value)

				// Randomly perform get operations
				if rand.Intn(2) == 0 {
					randKey := fmt.Sprintf("key%d-%d", id, rand.Intn(j+1))
					_, _ = k.Get(randKey)
				}
			}
		}(i)
	}

	wg.Wait()

	if k.QueueLen() > k.Capacity {
		t.Errorf("Cache exceeded capacity. Length: %d, Capacity: %d", k.QueueLen(), k.Capacity)
	}
}

func TestConcurrentSetWithTTL(t *testing.T) {
	k := NewCache(1000, time.Second*5)
	var wg sync.WaitGroup
	numOps := 10
	numGoroutines := 2

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := fmt.Sprintf("key%d-%d", id, j)
				value := fmt.Sprintf("value%d-%d", id, j)
				ttl := time.Duration(rand.Intn(3)+1) * time.Second
				k.Set(key, value, ttl)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(30 * time.Second) // Wait for all items to expire

	if k.QueueLen() != 0 {
		t.Errorf("Cache should be empty after TTL expiration. Length: %d", k.QueueLen())
	}
}

func TestConcurrentGetAndRemove(t *testing.T) {
	k := NewCache(1000, time.Second*5)
	var wg sync.WaitGroup
	numOps := 10
	numGoroutines := 2

	// Prefill the cache
	for i := 0; i < numOps; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		k.Set(key, value)
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := fmt.Sprintf("key%d", rand.Intn(numOps))
				if rand.Intn(2) == 0 {
					_, _ = k.Get(key)
				} else {
					if tmp, ok := k.SMap.Load(key); ok {
						node := tmp.(*Node)
						k.Remove(node)
					}
				}
			}
		}()
	}

	wg.Wait()

	// Verify that the cache is still in a consistent state
	if lenSyncMap(&k.SMap) != k.Queue.Len {
		t.Errorf("Inconsistent cache state. SMap length: %d, Queue length: %d", lenSyncMap(&k.SMap), k.Queue.Len)
	}
}
