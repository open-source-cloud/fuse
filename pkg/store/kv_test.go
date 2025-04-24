package store_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/pkg/store"
	"github.com/stretchr/testify/suite"
)

type KvTestSuit struct {
	suite.Suite
}

func TestKvStore(t *testing.T) {
	suite.Run(t, new(KvTestSuit))
}

func (s *KvTestSuit) TestKvStore() {
	kv := store.New()
	kv.Set("key", "value")
	s.Equal("value", kv.Get("key"))
}

// TestBasicOperations tests basic CRUD operations
func (s *KvTestSuit) TestBasicOperations() {
	kv := store.New()

	// Test Set and Get
	kv.Set("string", "value")
	s.Equal("value", kv.Get("string"))

	// Test various data types
	kv.Set("int", 42)
	s.Equal(42, kv.Get("int"))

	kv.Set("bool", true)
	s.Equal(true, kv.Get("bool"))

	kv.Set("float", 3.14)
	s.Equal(3.14, kv.Get("float"))

	// Test Has
	s.True(kv.Has("string"))
	s.False(kv.Has("nonexistent"))

	// Test Delete
	kv.Delete("string")
	s.Nil(kv.Get("string"))
	s.False(kv.Has("string"))
}

// TestTypedGetters tests the typed getter methods
func (s *KvTestSuit) TestTypedGetters() {
	kv := store.New()

	// String getter
	kv.Set("string", "value")
	s.Equal("value", kv.GetStr("string"))
	s.Equal("", kv.GetStr("nonexistent"))

	// Int getter
	kv.Set("int", 42)
	s.Equal(42, kv.GetInt("int"))
	s.Equal(0, kv.GetInt("nonexistent"))

	// Bool getter
	kv.Set("bool", true)
	s.Equal(true, kv.GetBool("bool"))
	s.Equal(false, kv.GetBool("nonexistent"))

	// Float getter
	kv.Set("float", 3.14)
	s.Equal(3.14, kv.GetFloat("float"))
	s.Equal(0.0, kv.GetFloat("nonexistent"))
}

// TestConcurrentAccess tests concurrent access to the KV store
func (s *KvTestSuit) TestConcurrentAccess() {
	kv := store.New()
	numGoroutines := 100
	numOperations := 100
	var wg sync.WaitGroup

	// Add multiple keys concurrently
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				kv.Set(key, fmt.Sprintf("value-%d-%d", id, j))
			}
		}(i)
	}
	wg.Wait()

	// Verify all keys were stored correctly
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < numOperations; j++ {
			key := fmt.Sprintf("key-%d-%d", i, j)
			expectedValue := fmt.Sprintf("value-%d-%d", i, j)
			s.Equal(expectedValue, kv.Get(key))
		}
	}
}

// TestConcurrentReadWrite tests concurrent reads and writes
func (s *KvTestSuit) TestConcurrentReadWrite() {
	kv := store.New()
	const numKeys = 100

	// Pre-populate some keys
	for i := 0; i < numKeys; i++ {
		kv.Set(fmt.Sprintf("key-%d", i), i)
	}

	var wg sync.WaitGroup
	numReaders := 10
	numWriters := 10
	wg.Add(numReaders + numWriters)

	// Start readers
	for i := 0; i < numReaders; i++ {
		go func(id int) {
			defer wg.Done()
			// go:lint
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))
			for j := 0; j < 1000; j++ {
				keyID := r.Intn(numKeys)
				key := fmt.Sprintf("key-%d", keyID)
				value := kv.Get(key)
				// No assertion, just testing concurrent access
				_ = value
			}
		}(i)
	}

	// Start writers
	for i := 0; i < numWriters; i++ {
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id+100)))
			for j := 0; j < 1000; j++ {
				keyID := r.Intn(numKeys)
				key := fmt.Sprintf("key-%d", keyID)
				kv.Set(key, fmt.Sprintf("writer-%d-value-%d", id, j))
			}
		}(i)
	}

	wg.Wait()
	// No assertion needed - just making sure it doesn't deadlock or panic
}

// TestConcurrentMixedOperations tests a mix of Get, Set, Has, and Delete
func (s *KvTestSuit) TestConcurrentMixedOperations() {
	kv := store.New()
	const numKeys = 100

	// Pre-populate some keys
	for i := 0; i < numKeys; i++ {
		kv.Set(fmt.Sprintf("key-%d", i), i)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))

			for j := 0; j < 1000; j++ {
				keyID := r.Intn(numKeys)
				key := fmt.Sprintf("key-%d", keyID)

				// Mix of operations
				op := r.Intn(4)
				switch op {
				case 0: // Get
					_ = kv.Get(key)
				case 1: // Set
					kv.Set(key, fmt.Sprintf("thread-%d-value-%d", id, j))
				case 2: // Has
					_ = kv.Has(key)
				case 3: // Delete
					kv.Delete(key)
					// Recreate key to ensure we don't run out of keys
					if r.Intn(2) == 0 {
						kv.Set(key, fmt.Sprintf("recreated-%d", j))
					}
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestTypedConcurrentAccess tests concurrent access with typed getters
func (s *KvTestSuit) TestTypedConcurrentAccess() {
	kv := store.New()

	// Pre-populate with different types
	for i := 0; i < 100; i++ {
		kv.Set(fmt.Sprintf("int-%d", i), i)
		kv.Set(fmt.Sprintf("str-%d", i), fmt.Sprintf("value-%d", i))
		kv.Set(fmt.Sprintf("bool-%d", i), i%2 == 0)
		kv.Set(fmt.Sprintf("float-%d", i), float64(i)/10.0)
	}

	var wg sync.WaitGroup
	numGoroutines := 20
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))

			for j := 0; j < 500; j++ {
				keyID := r.Intn(100)

				// Mix of typed getters
				op := r.Intn(4)
				switch op {
				case 0:
					_ = kv.GetStr(fmt.Sprintf("str-%d", keyID))
				case 1:
					_ = kv.GetInt(fmt.Sprintf("int-%d", keyID))
				case 2:
					_ = kv.GetBool(fmt.Sprintf("bool-%d", keyID))
				case 3:
					_ = kv.GetFloat(fmt.Sprintf("float-%d", keyID))
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestSingleKeyHighContention tests concurrent access to a single key
func (s *KvTestSuit) TestSingleKeyHighContention() {
	kv := store.New()
	const testKey = "high-contention-key"
	kv.Set(testKey, 0)

	var wg sync.WaitGroup
	numWriters := 100
	wg.Add(numWriters)

	for i := 0; i < numWriters; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				kv.Set(testKey, id*1000+j)
				time.Sleep(time.Microsecond) // Introduce small timing variations
				_ = kv.Get(testKey)
			}
		}(i)
	}

	wg.Wait()

	// Verify the key exists (we don't care about the value as it will be one of the last written)
	s.True(kv.Has(testKey))
}

// BenchmarkOperations benchmarks basic operations
func BenchmarkOperations(b *testing.B) {
	b.Run("Set", func(b *testing.B) {
		kv := store.New()
		for i := 0; i < b.N; i++ {
			kv.Set(fmt.Sprintf("key-%d", i), i)
		}
	})

	b.Run("Get", func(b *testing.B) {
		kv := store.New()
		// Prepare data
		for i := 0; i < 1000; i++ {
			kv.Set(fmt.Sprintf("key-%d", i), i)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			kv.Get(fmt.Sprintf("key-%d", i%1000))
		}
	})

	b.Run("Has", func(b *testing.B) {
		kv := store.New()
		// Prepare data
		for i := 0; i < 1000; i++ {
			kv.Set(fmt.Sprintf("key-%d", i), i)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			kv.Has(fmt.Sprintf("key-%d", i%1000))
		}
	})

	b.Run("Delete", func(b *testing.B) {
		kv := store.New()
		keys := make([]string, b.N)
		// Prepare keys
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i)
			keys[i] = key
			kv.Set(key, i)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			kv.Delete(keys[i])
		}
	})
}

// BenchmarkTypedGetters benchmarks typed getter methods
func BenchmarkTypedGetters(b *testing.B) {
	kv := store.New()
	// Prepare data
	for i := 0; i < 1000; i++ {
		kv.Set(fmt.Sprintf("int-%d", i), i)
		kv.Set(fmt.Sprintf("str-%d", i), fmt.Sprintf("value-%d", i))
		kv.Set(fmt.Sprintf("bool-%d", i), i%2 == 0)
		kv.Set(fmt.Sprintf("float-%d", i), float64(i)/10.0)
	}

	b.Run("GetStr", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			kv.GetStr(fmt.Sprintf("str-%d", i%1000))
		}
	})

	b.Run("GetInt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			kv.GetInt(fmt.Sprintf("int-%d", i%1000))
		}
	})

	b.Run("GetBool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			kv.GetBool(fmt.Sprintf("bool-%d", i%1000))
		}
	})

	b.Run("GetFloat", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			kv.GetFloat(fmt.Sprintf("float-%d", i%1000))
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent operations
func BenchmarkConcurrentOperations(b *testing.B) {
	// Test different concurrency levels
	for threads := 1; threads <= 8; threads *= 2 {
		b.Run(fmt.Sprintf("Set_%dThreads", threads), func(b *testing.B) {
			kv := store.New()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := fmt.Sprintf("key-%d", i)
					kv.Set(key, i)
					i++
				}
			})
		})

		b.Run(fmt.Sprintf("Get_%dThreads", threads), func(b *testing.B) {
			kv := store.New()
			// Prepare data
			for i := 0; i < 10000; i++ {
				kv.Set(fmt.Sprintf("key-%d", i), i)
			}
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := fmt.Sprintf("key-%d", i%10000)
					kv.Get(key)
					i++
				}
			})
		})

		b.Run(fmt.Sprintf("Mixed_%dThreads", threads), func(b *testing.B) {
			kv := store.New()
			// Prepare initial data
			for i := 0; i < 1000; i++ {
				kv.Set(fmt.Sprintf("key-%d", i), i)
			}
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				i := 0
				for pb.Next() {
					key := fmt.Sprintf("key-%d", i%1000)
					if r.Intn(10) < 3 { // 30% writes, 70% reads
						kv.Set(key, i)
					} else {
						kv.Get(key)
					}
					i++
				}
			})
		})
	}
}
