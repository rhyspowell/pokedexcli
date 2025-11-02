package pokecache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestAdd(t *testing.T) {
	cache := NewCache(5 * time.Second)
	key := "test-key"
	val := []byte("test-value")

	cache.Add(key, val)

	got, ok := cache.Get(key)
	if !ok {
		t.Fatal("expected to find key in cache")
	}
	if string(got) != string(val) {
		t.Errorf("expected %s, got %s", string(val), string(got))
	}
}

func TestGet(t *testing.T) {
	cache := NewCache(5 * time.Second)

	// Test getting non-existent key
	_, ok := cache.Get("non-existent")
	if ok {
		t.Error("expected false for non-existent key")
	}

	// Test getting existing key
	key := "test-key"
	val := []byte("test-value")
	cache.Add(key, val)

	got, ok := cache.Get(key)
	if !ok {
		t.Fatal("expected to find key in cache")
	}
	if string(got) != string(val) {
		t.Errorf("expected %s, got %s", string(val), string(got))
	}
}

func TestReapLoop(t *testing.T) {
	interval := 100 * time.Millisecond
	cache := NewCache(interval)

	// Add an entry
	key := "test-key"
	val := []byte("test-value")
	cache.Add(key, val)

	// Verify it exists
	_, ok := cache.Get(key)
	if !ok {
		t.Fatal("expected to find key in cache immediately after adding")
	}

	// Wait for the entry to expire (interval + small buffer)
	time.Sleep(interval + 50*time.Millisecond)

	// Verify it's been removed
	_, ok = cache.Get(key)
	if ok {
		t.Error("expected entry to be reaped after interval")
	}
}

func TestMultipleEntries(t *testing.T) {
	cache := NewCache(5 * time.Second)

	// Add multiple entries
	cache.Add("key1", []byte("value1"))
	cache.Add("key2", []byte("value2"))
	cache.Add("key3", []byte("value3"))

	// Verify all entries exist
	if val, ok := cache.Get("key1"); !ok || string(val) != "value1" {
		t.Error("expected key1 to exist with value1")
	}
	if val, ok := cache.Get("key2"); !ok || string(val) != "value2" {
		t.Error("expected key2 to exist with value2")
	}
	if val, ok := cache.Get("key3"); !ok || string(val) != "value3" {
		t.Error("expected key3 to exist with value3")
	}
}

func TestConcurrentAccess(t *testing.T) {
	cache := NewCache(5 * time.Second)
	numGoroutines := 10
	numOperations := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // One for writes, one for reads

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				val := []byte(fmt.Sprintf("value-%d-%d", id, j))
				cache.Add(key, val)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()
	// If we get here without a race condition, the test passed
}
