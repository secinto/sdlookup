package api

import (
	"testing"
	"time"

	"github.com/h4sh5/sdlookup/internal/models"
)

func TestLRUCache_GetSet(t *testing.T) {
	cache := NewLRUCache(10, time.Hour)

	info := &models.ShodanIPInfo{
		IP:    "192.168.1.1",
		Ports: []int{80, 443},
	}

	// Test Get on empty cache
	if _, ok := cache.Get("192.168.1.1"); ok {
		t.Error("Expected cache miss, got hit")
	}

	// Test Set and Get
	cache.Set("192.168.1.1", info)

	retrieved, ok := cache.Get("192.168.1.1")
	if !ok {
		t.Fatal("Expected cache hit, got miss")
	}

	if retrieved.IP != info.IP {
		t.Errorf("Retrieved IP = %s, want %s", retrieved.IP, info.IP)
	}
}

func TestLRUCache_Expiration(t *testing.T) {
	cache := NewLRUCache(10, 100*time.Millisecond)

	info := &models.ShodanIPInfo{
		IP:    "192.168.1.1",
		Ports: []int{80},
	}

	cache.Set("192.168.1.1", info)

	// Should be in cache
	if _, ok := cache.Get("192.168.1.1"); !ok {
		t.Error("Expected cache hit before expiration")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	if _, ok := cache.Get("192.168.1.1"); ok {
		t.Error("Expected cache miss after expiration")
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	cache := NewLRUCache(3, time.Hour)

	// Fill cache
	for i := 1; i <= 3; i++ {
		info := &models.ShodanIPInfo{
			IP:    "192.168.1." + string(rune('0'+i)),
			Ports: []int{80},
		}
		cache.Set(info.IP, info)
	}

	if cache.Size() != 3 {
		t.Errorf("Cache size = %d, want 3", cache.Size())
	}

	// Add one more - should evict oldest
	info := &models.ShodanIPInfo{
		IP:    "192.168.1.4",
		Ports: []int{80},
	}
	cache.Set(info.IP, info)

	if cache.Size() != 3 {
		t.Errorf("Cache size = %d, want 3", cache.Size())
	}

	// First entry should be evicted
	if _, ok := cache.Get("192.168.1.1"); ok {
		t.Error("Expected first entry to be evicted")
	}

	// Last entry should exist
	if _, ok := cache.Get("192.168.1.4"); !ok {
		t.Error("Expected last entry to exist")
	}
}

func TestLRUCache_Update(t *testing.T) {
	cache := NewLRUCache(10, time.Hour)

	info1 := &models.ShodanIPInfo{
		IP:    "192.168.1.1",
		Ports: []int{80},
	}

	info2 := &models.ShodanIPInfo{
		IP:    "192.168.1.1",
		Ports: []int{80, 443},
	}

	cache.Set("192.168.1.1", info1)
	cache.Set("192.168.1.1", info2)

	retrieved, ok := cache.Get("192.168.1.1")
	if !ok {
		t.Fatal("Expected cache hit")
	}

	if len(retrieved.Ports) != 2 {
		t.Errorf("Updated cache entry has %d ports, want 2", len(retrieved.Ports))
	}
}

func TestLRUCache_Clear(t *testing.T) {
	cache := NewLRUCache(10, time.Hour)

	for i := 1; i <= 5; i++ {
		info := &models.ShodanIPInfo{
			IP:    "192.168.1." + string(rune('0'+i)),
			Ports: []int{80},
		}
		cache.Set(info.IP, info)
	}

	if cache.Size() != 5 {
		t.Errorf("Cache size = %d, want 5", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Cache size after clear = %d, want 0", cache.Size())
	}
}

func TestLRUCache_Concurrent(t *testing.T) {
	cache := NewLRUCache(100, time.Hour)

	done := make(chan bool)

	// Multiple goroutines writing
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				info := &models.ShodanIPInfo{
					IP:    "192.168.1.1",
					Ports: []int{80},
				}
				cache.Set(info.IP, info)
			}
			done <- true
		}(i)
	}

	// Multiple goroutines reading
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				cache.Get("192.168.1.1")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Should not panic
}
