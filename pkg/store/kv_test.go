package store_test

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/uuid"
	"github.com/rs/zerolog/log"
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
			// nolint:gosec
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
			// nolint:gosec
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
			// nolint:gosec
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
			// nolint:gosec
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

// TestWithDotNotation tests dot notation support
func (s *KvTestSuit) TestWithDotNotation() {
	kv := store.New()
	kv.Set("a.b.c", "value")
	s.Equal("value", kv.Get("a.b.c"))

	// Set up a complex nested structure
	kv.Set("user.profile.name", "John Doe")
	kv.Set("user.profile.age", 30)
	kv.Set("user.profile.active", true)
	kv.Set("user.address.city", "New York")
	kv.Set("user.address.zip", 10001)
	kv.Set("user.scores", []int{95, 88, 92})
	kv.Set("app.settings.theme.dark", true)
	kv.Set("app.settings.theme.fontSize", 16)
	kv.Set("app.settings.notifications.email", false)
	kv.Set("app.version", 2.1)

	tests := []struct {
		key           string
		expectedValue any
	}{
		// Test simple dot notation
		{"a.b.c", "value"},

		// Test user profile fields
		{"user.profile.name", "John Doe"},
		{"user.profile.age", 30},
		{"user.profile.active", true},

		// Test nested address fields
		{"user.address.city", "New York"},
		{"user.address.zip", 10001},

		// Test array access (though objx doesn't support array indexing in dot notation directly)
		{"user.scores", []int{95, 88, 92}},

		// Test deeply nested fields
		{"app.settings.theme.dark", true},
		{"app.settings.theme.fontSize", 16},
		{"app.settings.notifications.email", false},
		{"app.version", 2.1},

		// Test non-existent keys
		{"user.profile.email", nil},
		{"nonexistent.key", nil},
		{"app.settings.theme.nonexistent", nil},

		// Test parent paths
		{"user.profile", map[string]any{
			"name":   "John Doe",
			"age":    30,
			"active": true,
		}},
		{"user", map[string]any{
			"profile": map[string]any{
				"name":   "John Doe",
				"age":    30,
				"active": true,
			},
			"address": map[string]any{
				"city": "New York",
				"zip":  10001,
			},
			"scores": []int{95, 88, 92},
		}},
		{"app.settings", map[string]any{
			"theme": map[string]any{
				"dark":     true,
				"fontSize": 16,
			},
			"notifications": map[string]any{
				"email": false,
			},
		}},
	}

	for _, test := range tests {
		s.Run(test.key, func() {
			actual := kv.Get(test.key)

			// Handle map comparisons specially since they can be tricky to compare directly
			if mapVal, ok := test.expectedValue.(map[string]any); ok {
				actualMap, actualOk := actual.(map[string]any)
				s.True(actualOk, "Expected a map for key %s, got: %T", test.key, actual)

				// Compare map sizes
				s.Equal(len(mapVal), len(actualMap), "Map sizes don't match for key %s", test.key)

				// Compare map contents
				for k, v := range mapVal {
					s.Contains(actualMap, k, "Map for key %s missing expected key %s", test.key, k)
					// For simplicity, we don't do deep comparisons of nested maps
					if innerMap, isMap := v.(map[string]any); isMap {
						s.IsType(innerMap, actualMap[k], "Types don't match for nested map at %s.%s", test.key, k)
					} else {
						s.Equal(v, actualMap[k], "Values don't match for %s.%s", test.key, k)
					}
				}
			} else {
				// For non-map values, use regular Equal
				s.Equal(test.expectedValue, actual, "Values don't match for key %s", test.key)
			}
		})
	}

	// Test typed getters with dot notation
	s.Equal("John Doe", kv.GetStr("user.profile.name"))
	s.Equal(30, kv.GetInt("user.profile.age"))
	s.Equal(true, kv.GetBool("user.profile.active"))
	s.Equal(2.1, kv.GetFloat("app.version"))

	// TODO: Test deleting with dot notation
	// kv.Delete("user.profile.name")
	// s.Nil(kv.Get("user.profile.name"))
	// s.NotNil(kv.Get("user.profile"))

	// Test overwriting
	kv.Set("app.settings", "overwritten")
	s.Equal("overwritten", kv.Get("app.settings"))
	s.Nil(kv.Get("app.settings.theme.dark")) // This should now be nil as we overwrote the parent
}

// TestDotNotationWithArrays tests the use of dot notation and indexed access for handling arrays and nested data structures.
func (s *KvTestSuit) TestDotNotationWithArrays() {
	kv := store.New()

	// Set up arrays at different nesting levels
	kv.Set("simple", []string{"a", "b", "c"})
	kv.Set("numbers", []int{1, 2, 3, 4, 5})
	kv.Set("mixed", []interface{}{1, "two", true, 4.5})

	// Set up nested structures with arrays
	kv.Set("user.hobbies", []string{"reading", "coding", "hiking"})
	kv.Set("user.scores.math", []int{95, 87, 92})
	kv.Set("user.scores.science", []int{88, 91, 94})

	// Set up an array of objects
	kv.Set("products", []map[string]interface{}{
		{"id": 1, "name": "Product 1", "price": 19.99},
		{"id": 2, "name": "Product 2", "price": 29.99},
		{"id": 3, "name": "Product 3", "price": 39.99},
	})

	// Set up a more complex nested structure with arrays
	kv.Set("app.config.allowedTypes", []string{"jpg", "png", "gif"})
	kv.Set("app.users", []map[string]interface{}{
		{
			"id":   1,
			"name": "John",
			"tags": []string{"admin", "active"},
			"sessions": []map[string]interface{}{
				{"id": "abc", "lastLogin": "2023-01-01"},
				{"id": "def", "lastLogin": "2023-01-15"},
			},
		},
		{
			"id":   2,
			"name": "Jane",
			"tags": []string{"user", "active"},
			"sessions": []map[string]interface{}{
				{"id": "ghi", "lastLogin": "2023-01-10"},
			},
		},
	})

	// Test multi-dimensional arrays
	kv.Set("matrix", [][]int{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	})

	// Test simple array access
	simpleArray := kv.Get("simple")
	s.IsType([]string{}, simpleArray)
	s.Equal([]string{"a", "b", "c"}, simpleArray)

	// Test array index access using dot notation
	s.Equal("a", kv.Get("simple[0]"))
	s.Equal("b", kv.Get("simple[1]"))
	s.Equal("c", kv.Get("simple[2]"))
	s.Nil(kv.Get("simple[3]"))  // Out of bounds
	s.Nil(kv.Get("simple[-1]")) // Negative index

	// Test number array with index access
	s.Equal(1, kv.Get("numbers[0]"))
	s.Equal(3, kv.Get("numbers[2]"))
	s.Equal(5, kv.Get("numbers[4]"))

	// Test typed getters with an array index
	s.Equal("a", kv.GetStr("simple[0]"))
	s.Equal(1, kv.GetInt("numbers[0]"))
	s.Equal(true, kv.GetBool("mixed[2]"))
	s.Equal(4.5, kv.GetFloat("mixed[3]"))

	// Test nested array access
	s.Equal("reading", kv.Get("user.hobbies[0]"))
	s.Equal("hiking", kv.Get("user.hobbies[2]"))
	s.Equal(95, kv.Get("user.scores.math[0]"))
	s.Equal(94, kv.Get("user.scores.science[2]"))

	// Test access to an array of objects
	s.Equal(1, kv.Get("products[0].id"))
	s.Equal("Product 2", kv.Get("products[1].name"))
	s.Equal(39.99, kv.Get("products[2].price"))

	// Test deeply nested structures with arrays
	s.Equal("jpg", kv.Get("app.config.allowedTypes[0]"))
	s.Equal("gif", kv.Get("app.config.allowedTypes[2]"))

	// Test array of objects with deep nesting
	s.Equal(1, kv.Get("app.users[0].id"))
	s.Equal("Jane", kv.Get("app.users[1].name"))
	s.Equal("admin", kv.Get("app.users[0].tags[0]"))
	s.Equal("active", kv.Get("app.users[1].tags[1]"))

	// Test multi-level array indexing
	s.Equal("abc", kv.Get("app.users[0].sessions[0].id"))
	s.Equal("2023-01-15", kv.Get("app.users[0].sessions[1].lastLogin"))
	s.Equal("ghi", kv.Get("app.users[1].sessions[0].id"))

	// Test multi-dimensional array access
	s.Equal(1, kv.Get("matrix[0][0]"))
	s.Equal(5, kv.Get("matrix[1][1]"))
	s.Equal(9, kv.Get("matrix[2][2]"))

	kv.Set("products[1].price", 99.99)
	s.Equal(99.99, kv.Get("products[1].price"))

	kv.Set("app.users[0].tags[1]", "super-admin")
	s.Equal("super-admin", kv.Get("app.users[0].tags[1]"))

	// Test modifying deep structures
	kv.Set("app.users[1].sessions[0].lastLogin", "2023-02-01")
	s.Equal("2023-02-01", kv.Get("app.users[1].sessions[0].lastLogin"))

	// Test adding new elements to existing objects via dot notation
	kv.Set("products[1].inStock", true)
	s.Equal(true, kv.Get("products[1].inStock"))

	// Test boundary cases
	kv.Set("empty", []string{})
	s.Nil(kv.Get("empty[0]"))

	// Test with non-array values
	kv.Set("scalar", "value")
	s.Nil(kv.Get("scalar[0].a"))

	// Test overwriting array elements with objects
	kv.Set("numbers[2]", map[string]interface{}{"value": 42})
	s.Equal(42, kv.Get("numbers[2].value"))

	// Test creating arrays via indexed notation
	kv.Set("newArray[0]", "first")
	kv.Set("newArray[1]", "second")

	// Test with very large indices
	s.Nil(kv.Get("simple[999]"))

	// Test with a string that looks like an index but isn't a valid index
	kv.Set("keyWithDot", "value")
	kv.Set("keyWithDot.123abc", "another value")
	s.Equal("another value", kv.Get("keyWithDot.123abc"))

	// Test nested indices after a modified array
	kv.Set("nestedArrays", []interface{}{
		[]int{1, 2, 3},
		[]string{"a", "b", "c"},
	})
	s.Equal(2, kv.Get("nestedArrays[0][1]"))
	s.Equal("b", kv.Get("nestedArrays[1][1]"))

	// Modify a nested array and test again
	kv.Set("nestedArrays[1][1]", "MODIFIED")
	s.Equal("MODIFIED", kv.Get("nestedArrays[1][1]"))

	// Test with mixed map and array access
	kv.Set("complexMix", map[string]interface{}{
		"users": []map[string]interface{}{
			{"name": "Alice", "scores": []int{90, 85, 88}},
			{"name": "Bob", "scores": []int{78, 92, 86}},
		},
	})
	s.Equal("Alice", kv.Get("complexMix.users[0].name"))
	s.Equal(85, kv.Get("complexMix.users[0].scores[1]"))
	s.Equal(86, kv.Get("complexMix.users[1].scores[2]"))

	// Update nested values
	kv.Set("complexMix.users[1].scores[0]", 100)
	s.Equal(100, kv.Get("complexMix.users[1].scores[0]"))
}

// TestDotNotationWithMaps tests the use of dot notation and map access for handling maps and nested data structures.
func (s *KvTestSuit) TestDotNotationWithMaps() {
	kv := store.New()

	// Set up a simple map
	kv.Set("config", map[string]interface{}{
		"port":    8080,
		"host":    "localhost",
		"debug":   true,
		"timeout": 30.5,
	})

	// Set up nested maps
	kv.Set("database", map[string]interface{}{
		"primary": map[string]interface{}{
			"host":     "db.example.com",
			"port":     5432,
			"username": "admin",
			"password": "secret",
			"settings": map[string]interface{}{
				"maxConnections": 100,
				"timeout":        5.0,
				"ssl":            true,
			},
		},
		"replica": map[string]interface{}{
			"host":     "replica.example.com",
			"port":     5432,
			"username": "reader",
			"password": "readonly",
		},
	})

	// Test direct access to map properties
	s.Equal(8080, kv.Get("config.port"))
	s.Equal("localhost", kv.Get("config.host"))
	s.Equal(true, kv.Get("config.debug"))
	s.Equal(30.5, kv.Get("config.timeout"))

	// Test nested map access
	s.Equal("db.example.com", kv.Get("database.primary.host"))
	s.Equal(5432, kv.Get("database.primary.port"))
	s.Equal("admin", kv.Get("database.primary.username"))
	s.Equal("secret", kv.Get("database.primary.password"))

	// Test deeply nested map access
	s.Equal(100, kv.Get("database.primary.settings.maxConnections"))
	s.Equal(5.0, kv.Get("database.primary.settings.timeout"))
	s.Equal(true, kv.Get("database.primary.settings.ssl"))

	// Test access to the second nested map
	s.Equal("replica.example.com", kv.Get("database.replica.host"))
	s.Equal("reader", kv.Get("database.replica.username"))

	// Test typed getters with map access
	s.Equal("localhost", kv.GetStr("config.host"))
	s.Equal(8080, kv.GetInt("config.port"))
	s.Equal(true, kv.GetBool("config.debug"))
	s.Equal(30.5, kv.GetFloat("config.timeout"))

	// Test modifying nested map values
	kv.Set("database.primary.port", 3306)
	s.Equal(3306, kv.Get("database.primary.port"))

	kv.Set("database.primary.settings.maxConnections", 200)
	s.Equal(200, kv.Get("database.primary.settings.maxConnections"))

	// Test adding new properties to existing maps
	kv.Set("config.newSetting", "value")
	s.Equal("value", kv.Get("config.newSetting"))

	kv.Set("database.primary.settings.newOption", false)
	s.Equal(false, kv.Get("database.primary.settings.newOption"))

	// Test creating new nested maps via dot notation
	kv.Set("newMap.nested.deeply.value", 42)
	s.Equal(42, kv.Get("newMap.nested.deeply.value"))

	// Test overwriting a map with a scalar value
	kv.Set("database.replica", "overwritten")
	s.Equal("overwritten", kv.Get("database.replica"))
	s.Nil(kv.Get("database.replica.host")) // Should now be nil since parent was overwritten

	// Test mixing map and array notation
	kv.Set("servers", map[string]interface{}{
		"production": []map[string]interface{}{
			{"host": "prod1.example.com", "region": "us-east"},
			{"host": "prod2.example.com", "region": "us-west"},
		},
		"staging": []map[string]interface{}{
			{"host": "stage.example.com", "region": "eu-central"},
		},
	})

	s.Equal("prod1.example.com", kv.Get("servers.production[0].host"))
	s.Equal("us-west", kv.Get("servers.production[1].region"))
	s.Equal("eu-central", kv.Get("servers.staging[0].region"))

	// Test modifying values in mixed map/array structures
	kv.Set("servers.production[0].host", "new-prod1.example.com")
	s.Equal("new-prod1.example.com", kv.Get("servers.production[0].host"))

	// Test boundary cases
	s.Nil(kv.Get("config.nonexistent"))
	s.Nil(kv.Get("database.primary.nonexistent"))
	s.Nil(kv.Get("nonexistent.path"))

	// Test with a complex nested structure containing different data types
	kv.Set("complex", map[string]interface{}{
		"string": "value",
		"int":    42,
		"bool":   true,
		"float":  3.14,
		"array":  []int{1, 2, 3},
		"map": map[string]interface{}{
			"nested": "nested_value",
		},
		"mixedArray": []interface{}{
			"string",
			42,
			map[string]interface{}{"key": "value"},
		},
	})

	s.Equal("value", kv.Get("complex.string"))
	s.Equal(42, kv.Get("complex.int"))
	s.Equal(true, kv.Get("complex.bool"))
	s.Equal(3.14, kv.Get("complex.float"))
	s.Equal([]int{1, 2, 3}, kv.Get("complex.array"))
	s.Equal("nested_value", kv.Get("complex.map.nested"))
	s.Equal("string", kv.Get("complex.mixedArray[0]"))
	s.Equal(42, kv.Get("complex.mixedArray[1]"))
	s.Equal("value", kv.Get("complex.mixedArray[2].key"))
}

// TestWithUUID validates the storage and retrieval of UUID keys and values in a key-value store, including nested structures.
func (s *KvTestSuit) TestWithUUID() {
	kv := store.New()

	id := uuid.V7()

	kv.Set("id", id)
	s.Equal(id, kv.Get("id"))

	nestedId := uuid.V7()
	kv.Set("nested.id", nestedId)
	s.Equal(nestedId, kv.Get("nested.id"))

	mapVal := map[string]any{
		nestedId: uuid.V7(),
	}
	uuidInKey := fmt.Sprintf("nested.%s", nestedId)
	kv.Set(uuidInKey, mapVal)
	s.Equal(mapVal, kv.Get(uuidInKey))

	uuidInKeys := []string{
		uuid.V7(),
		uuid.V7(),
		uuid.V7(),
		uuid.V7(),
	}
	for _, uuidValue := range uuidInKeys {
		uuidInKeysNested := []string{
			uuid.V7(),
			uuid.V7(),
			uuid.V7(),
			uuid.V7(),
		}
		for _, uuidValueNested := range uuidInKeysNested {
			kv.Set(fmt.Sprintf("nested.%s.%s", uuidValue, uuidValueNested), uuidValueNested)
			s.Equal(uuidValueNested, kv.Get(fmt.Sprintf("nested.%s.%s", uuidValue, uuidValueNested)))
		}
	}

	log.Print(kv.Raw())
}

// TestMapForNodeWorkflow validates the correctness of storing and retrieving nested maps using dot notation in the KV store.
func (s *KvTestSuit) TestMapForNodeWorkflow() {
	kv := store.New()

	edges := []string{uuid.V7(), uuid.V7(), uuid.V7(), uuid.V7(), uuid.V7()}
	for _, edgeId := range edges {
		randVal := rand.Int()
		kv.Set(fmt.Sprintf("edges.%s.rand", edgeId), randVal)
	}

	for _, edgeId := range edges {
		randVal := kv.Get(fmt.Sprintf("edges.%s.rand", edgeId))
		s.NotNil(randVal)
	}
}

// @@ Benchmark @@

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
				// nolint:gosec
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
