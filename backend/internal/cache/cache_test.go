package cache

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ttl := 5 * time.Minute
	c := New(ttl)
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.ttl != ttl {
		t.Errorf("Expected TTL %v, got %v", ttl, c.ttl)
	}
}

func TestSetAndGet(t *testing.T) {
	c := New(5 * time.Minute)
	key := "testKey"
	value := "testValue"

	c.Set(key, value)
	retrieved, ok := c.Get(key)
	if !ok {
		t.Fatal("Get returned false for existing key")
	}
	if retrieved != value {
		t.Errorf("Expected %v, got %v", value, retrieved)
	}
}

func TestGetNonExistent(t *testing.T) {
	c := New(5 * time.Minute)
	_, ok := c.Get("nonExistent")
	if ok {
		t.Error("Get returned true for non-existent key")
	}
}

func TestGetExpired(t *testing.T) {
	c := New(1 * time.Millisecond) // Very short TTL
	key := "testKey"
	value := "testValue"

	c.Set(key, value)
	time.Sleep(2 * time.Millisecond) // Wait for expiration

	_, ok := c.Get(key)
	if ok {
		t.Error("Get returned true for expired key")
	}
}

func TestDelete(t *testing.T) {
	c := New(5 * time.Minute)
	key := "testKey"
	value := "testValue"

	c.Set(key, value)
	c.Delete(key)

	_, ok := c.Get(key)
	if ok {
		t.Error("Get returned true after Delete")
	}
}

func TestDeleteByPrefix(t *testing.T) {
	c := New(5 * time.Minute)

	// Set multiple keys
	c.Set("prefix1", "value1")
	c.Set("prefix2", "value2")
	c.Set("other", "value3")

	// Delete by prefix
	c.DeleteByPrefix("prefix")

	// Check that prefixed keys are gone
	_, ok1 := c.Get("prefix1")
	if ok1 {
		t.Error("prefix1 should be deleted")
	}
	_, ok2 := c.Get("prefix2")
	if ok2 {
		t.Error("prefix2 should be deleted")
	}

	// Check that non-prefixed key remains
	retrieved, ok3 := c.Get("other")
	if !ok3 || retrieved != "value3" {
		t.Error("other should remain")
	}
}
