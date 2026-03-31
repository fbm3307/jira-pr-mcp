package jira

import (
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	c := NewCache(1 * time.Minute)

	c.Set("key1", "value1")

	val, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}
	if val != "value1" {
		t.Errorf("got %v, want value1", val)
	}
}

func TestCache_Miss(t *testing.T) {
	c := NewCache(1 * time.Minute)

	_, ok := c.Get("nonexistent")
	if ok {
		t.Fatal("expected cache miss, got hit")
	}
}

func TestCache_Expiry(t *testing.T) {
	c := NewCache(1 * time.Millisecond)

	c.Set("key1", "value1")
	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("key1")
	if ok {
		t.Fatal("expected cache miss after expiry, got hit")
	}
}

func TestCache_Invalidate(t *testing.T) {
	c := NewCache(1 * time.Minute)

	c.Set("key1", "value1")
	c.Set("key2", "value2")

	c.Invalidate()

	if _, ok := c.Get("key1"); ok {
		t.Fatal("expected cache miss after invalidation for key1")
	}
	if _, ok := c.Get("key2"); ok {
		t.Fatal("expected cache miss after invalidation for key2")
	}
}
