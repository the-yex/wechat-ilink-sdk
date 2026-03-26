package t

import (
	"testing"
)

func TestMapStoreLoad(t *testing.T) {
	m := New[string, int]()

	// Test store and load
	m.Store("key1", 100)
	v, ok := m.Load("key1")
	if !ok || v != 100 {
		t.Errorf("Load() = %d, %v; want 100, true", v, ok)
	}

	// Test non-existent key
	v, ok = m.Load("notexist")
	if ok || v != 0 {
		t.Errorf("Load() = %d, %v; want 0, false", v, ok)
	}
}

func TestMapDelete(t *testing.T) {
	m := New[string, int]()

	m.Store("key1", 100)
	m.Delete("key1")

	v, ok := m.Load("key1")
	if ok {
		t.Errorf("Load() after Delete() = %d, %v; want 0, false", v, ok)
	}
}

func TestMapRange(t *testing.T) {
	m := New[string, int]()

	m.Store("a", 1)
	m.Store("b", 2)
	m.Store("c", 3)

	count := 0
	sum := 0
	m.Range(func(key string, value int) bool {
		count++
		sum += value
		return true
	})

	if count != 3 {
		t.Errorf("Range() visited %d items; want 3", count)
	}
	if sum != 6 {
		t.Errorf("Range() sum = %d; want 6", sum)
	}
}

func TestMapRangeStop(t *testing.T) {
	m := New[string, int]()

	m.Store("a", 1)
	m.Store("b", 2)
	m.Store("c", 3)

	count := 0
	m.Range(func(key string, value int) bool {
		count++
		return count < 2 // Stop after 2 items
	})

	if count != 2 {
		t.Errorf("Range() visited %d items; want 2", count)
	}
}

func TestMapLoadOrStore(t *testing.T) {
	m := New[string, int]()

	// First store
	v, loaded := m.LoadOrStore("key1", 100)
	if loaded || v != 100 {
		t.Errorf("LoadOrStore() = %d, %v; want 100, false", v, loaded)
	}

	// Second call should load existing
	v, loaded = m.LoadOrStore("key1", 200)
	if !loaded || v != 100 {
		t.Errorf("LoadOrStore() = %d, %v; want 100, true", v, loaded)
	}
}

func TestMapLoadAndDelete(t *testing.T) {
	m := New[string, int]()

	m.Store("key1", 100)

	// Load and delete
	v, loaded := m.LoadAndDelete("key1")
	if !loaded || v != 100 {
		t.Errorf("LoadAndDelete() = %d, %v; want 100, true", v, loaded)
	}

	// Should be deleted
	v, ok := m.Load("key1")
	if ok {
		t.Errorf("Load() after LoadAndDelete() = %d, %v; want 0, false", v, ok)
	}

	// Delete non-existent
	v, loaded = m.LoadAndDelete("notexist")
	if loaded {
		t.Errorf("LoadAndDelete() for non-existent = %d, %v; want 0, false", v, loaded)
	}
}

func TestMapGenericTypes(t *testing.T) {
	// Test with different types
	t.Run("int keys", func(t *testing.T) {
		m := New[int, string]()
		m.Store(1, "one")
		m.Store(2, "two")

		v, ok := m.Load(1)
		if !ok || v != "one" {
			t.Errorf("Load() = %s, %v; want 'one', true", v, ok)
		}
	})

	t.Run("struct values", func(t *testing.T) {
		type user struct {
			name string
			age  int
		}
		m := New[string, user]()
		m.Store("john", user{name: "John", age: 30})

		v, ok := m.Load("john")
		if !ok || v.name != "John" || v.age != 30 {
			t.Errorf("Load() = %+v, %v; want {name:John age:30}, true", v, ok)
		}
	})

	t.Run("pointer values", func(t *testing.T) {
		m := New[string, *int]()
		val := 42
		m.Store("answer", &val)

		v, ok := m.Load("answer")
		if !ok || v == nil || *v != 42 {
			t.Errorf("Load() = %v, %v; want *42, true", v, ok)
		}
	})
}

func TestMapConcurrent(t *testing.T) {
	m := New[int, int]()

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(n int) {
			m.Store(n, n*10)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 100; i++ {
		<-done
	}

	// Verify values
	for i := 0; i < 100; i++ {
		v, ok := m.Load(i)
		if !ok || v != i*10 {
			t.Errorf("Load(%d) = %d, %v; want %d, true", i, v, ok, i*10)
		}
	}
}