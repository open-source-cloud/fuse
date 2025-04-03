// nolint:gosec
package schema

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/suite"
)

// SchemaSuite defines the test suite for schema package
type SchemaSuite struct {
	suite.Suite
}

func TestSchemaSuite(t *testing.T) {
	suite.Run(t, new(SchemaSuite))
}

func (s *SchemaSuite) TestLoadSchemaFromJSON() {
	// Test valid schema
	validJSON := `{
		"id": "test-schema",
		"name": "Test Schema",
		"version": "1.0",
		"fields": [
			{
				"id": "field1",
				"name": "Field 1",
				"type": "string",
				"required": true
			}
		]
	}`

	schema, err := LoadSchemaFromJSON(validJSON)
	s.Require().NoError(err, "Failed to load valid schema")
	s.Equal("test-schema", schema.ID, "Schema ID should match")

	// Test invalid schema (missing required field)
	invalidJSON := `{
		"name": "Invalid Schema",
		"version": "1.0",
		"fields": []
	}`

	_, err = LoadSchemaFromJSON(invalidJSON)
	s.Error(err, "Should error for invalid schema")

	// Test invalid JSON
	_, err = LoadSchemaFromJSON("{invalid json")
	s.Error(err, "Should error for invalid JSON")
}

func (s *SchemaSuite) TestValidateSchemaDefinition() {
	// Test valid schema
	validSchema := &Schema{
		ID:      "test-schema",
		Name:    "Test Schema",
		Version: "1.0",
		Fields: []Field{
			{
				ID:       "field1",
				Name:     "Field 1",
				Type:     TypeString,
				Required: true,
			},
		},
	}

	err := ValidateSchemaDefinition(validSchema)
	s.NoError(err, "Valid schema should pass validation")

	// Test missing ID
	invalidSchema := &Schema{
		Name:    "Invalid Schema",
		Version: "1.0",
		Fields: []Field{
			{
				ID:       "field1",
				Name:     "Field 1",
				Type:     TypeString,
				Required: true,
			},
		},
	}

	err = ValidateSchemaDefinition(invalidSchema)
	s.Error(err, "Schema with missing ID should fail validation")

	// Test missing fields
	invalidSchema = &Schema{
		ID:      "invalid-schema",
		Name:    "Invalid Schema",
		Version: "1.0",
		Fields:  []Field{},
	}

	err = ValidateSchemaDefinition(invalidSchema)
	s.Error(err, "Schema with no fields should fail validation")

	// Test duplicate field IDs
	invalidSchema = &Schema{
		ID:      "invalid-schema",
		Name:    "Invalid Schema",
		Version: "1.0",
		Fields: []Field{
			{
				ID:       "field1",
				Name:     "Field 1",
				Type:     TypeString,
				Required: true,
			},
			{
				ID:       "field1", // Duplicate ID
				Name:     "Field 2",
				Type:     TypeString,
				Required: true,
			},
		},
	}

	err = ValidateSchemaDefinition(invalidSchema)
	s.Error(err, "Schema with duplicate field IDs should fail validation")
}

func (s *SchemaSuite) TestValidate() {
	// Create a schema for testing
	schema := Schema{
		ID:      "test-schema",
		Name:    "Test Schema",
		Version: "1.0",
		Fields: []Field{
			{
				ID:       "name",
				Name:     "Name",
				Type:     TypeString,
				Required: true,
				Validation: []ValidationRule{
					{
						Type:      "min_length",
						Value:     float64(3),
						Message:   "Name must be at least 3 characters long",
						ErrorCode: "name_too_short",
					},
				},
			},
			{
				ID:       "age",
				Name:     "Age",
				Type:     TypeInteger,
				Required: true,
				Validation: []ValidationRule{
					{
						Type:      "min",
						Value:     float64(18),
						Message:   "Must be at least 18 years old",
						ErrorCode: "age_too_young",
					},
				},
			},
			{
				ID:       "email",
				Name:     "Email",
				Type:     TypeString,
				Required: false,
				Validation: []ValidationRule{
					{
						Type:      "pattern",
						Value:     "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
						Message:   "Invalid email format",
						ErrorCode: "email_invalid_format",
					},
				},
			},
			{
				ID:       "preferences",
				Name:     "Preferences",
				Type:     TypeObject,
				Required: false,
				Properties: []Field{
					{
						ID:       "theme",
						Name:     "Theme",
						Type:     TypeString,
						Required: true,
						Validation: []ValidationRule{
							{
								Type:      "enum",
								Value:     []interface{}{"light", "dark"},
								Message:   "Theme must be either 'light' or 'dark'",
								ErrorCode: "theme_invalid",
							},
						},
					},
				},
			},
		},
	}

	// Test valid data
	validData := map[string]interface{}{
		"name":  "John Doe",
		"age":   float64(25),
		"email": "john@example.com",
		"preferences": map[string]interface{}{
			"theme": "dark",
		},
	}

	result := schema.Validate(validData)
	s.True(result.Valid, "Valid data should pass validation")
	s.Empty(result.Errors, "No errors should be present for valid data")

	// Test missing required field
	invalidData := map[string]interface{}{
		"name": "John Doe",
		// Missing 'age' field
	}

	result = schema.Validate(invalidData)
	s.False(result.Valid, "Missing required field should fail validation")

	// Test invalid field value (min_length)
	invalidData = map[string]interface{}{
		"name": "Jo", // Too short
		"age":  float64(25),
	}

	result = schema.Validate(invalidData)
	s.False(result.Valid, "Invalid name length should fail validation")

	// Test invalid field value (min_value)
	invalidData = map[string]interface{}{
		"name": "John Doe",
		"age":  float64(16), // Too young
	}

	result = schema.Validate(invalidData)
	s.False(result.Valid, "Invalid age should fail validation")

	// Test invalid pattern
	invalidData = map[string]interface{}{
		"name":  "John Doe",
		"age":   float64(25),
		"email": "not-an-email", // Invalid email format
	}

	result = schema.Validate(invalidData)
	s.False(result.Valid, "Invalid email should fail validation")

	// Test invalid nested object
	invalidData = map[string]interface{}{
		"name": "John Doe",
		"age":  float64(25),
		"preferences": map[string]interface{}{
			"theme": "blue", // Invalid theme
		},
	}

	result = schema.Validate(invalidData)
	s.False(result.Valid, "Invalid theme should fail validation")
}

// UserProfile represents a user profile for testing with faker
type UserProfile struct {
	Username    string      `faker:"username" json:"username"`
	Email       string      `faker:"email" json:"email"`
	Age         int         `faker:"boundary_start=18, boundary_end=90" json:"age"`
	Address     Address     `json:"address"`
	Preferences Preferences `json:"preferences"`
}

// Address represents a user address for testing
type Address struct {
	Street string `json:"street"`
	City   string `faker:"word" json:"city"`
	State  string `faker:"word" json:"state"`
	Zip    string `json:"zip"`
}

// Preferences represents user preferences for testing
type Preferences struct {
	Theme         string `json:"theme"`
	Notifications bool   `json:"notifications"`
}

// Product represents a product for testing with faker
type Product struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `faker:"sentence" json:"description"`
	Price       float64    `faker:"boundary_start=0.99, boundary_end=1000" json:"price"`
	Category    string     `json:"category"`
	Inventory   Inventory  `json:"inventory"`
	Dimensions  Dimensions `json:"dimensions"`
}

// Inventory represents product inventory for testing
type Inventory struct {
	Quantity  int    `faker:"boundary_start=0, boundary_end=1000" json:"quantity"`
	SKU       string `json:"sku"`
	Warehouse string `faker:"word" json:"warehouse"`
}

// Dimensions represents product dimensions for testing
type Dimensions struct {
	Length float64 `faker:"boundary_start=0.1, boundary_end=100" json:"length"`
	Width  float64 `faker:"boundary_start=0.1, boundary_end=100" json:"width"`
	Height float64 `faker:"boundary_start=0.1, boundary_end=100" json:"height"`
	Weight float64 `faker:"boundary_start=0.1, boundary_end=50" json:"weight"`
}

func (s *SchemaSuite) TestUserProfileSchemaWithFaker() {
	// Load the schema
	schemaFile := "../../examples/schemas/user_profile.json"
	schema, err := LoadSchemaFromFile(schemaFile)
	s.Require().NoError(err, "Failed to load schema")

	// Generate and test 5 random user profiles
	for i := 0; i < 5; i++ {
		// Generate a fake user profile
		user := UserProfile{}
		err := faker.FakeData(&user)
		s.Require().NoError(err, "Failed to generate fake data")

		// Manually set values that can't be generated by faker
		user.Preferences.Theme = generateRandomTheme()
		user.Preferences.Notifications = rand.Intn(2) == 1 // Random boolean
		user.Address.Street = fmt.Sprintf("%d %s", rand.Intn(999)+1, faker.Word())
		user.Address.Zip = fmt.Sprintf("%05d", rand.Intn(90000)+10000)

		// Convert to map for validation
		userData, err := structToMap(user)
		s.Require().NoError(err, "Failed to convert struct to map")

		// Validate against schema
		result := schema.Validate(userData)
		s.True(result.Valid, "User profile %s should be valid", user.Username)
	}

	// Test invalid data
	invalidUser := map[string]interface{}{
		"username": "a", // Too short
		"email":    "invalid-email",
		"age":      float64(10), // Too young
	}

	result := schema.Validate(invalidUser)
	s.False(result.Valid, "Invalid user data should fail validation")
}

func (s *SchemaSuite) TestProductSchemaWithFaker() {
	// Load the schema
	schemaFile := "../../examples/schemas/product.json"
	schema, err := LoadSchemaFromFile(schemaFile)
	s.Require().NoError(err, "Failed to load schema")

	// Generate and test 5 random products
	for i := 0; i < 5; i++ {
		// Generate a fake product
		product := Product{}
		err := faker.FakeData(&product)
		s.Require().NoError(err, "Failed to generate fake data")

		// Manually set values that can't be generated by faker
		product.ID = generateProductID()
		product.Name = generateProductName()
		product.Category = generateRandomCategory()
		product.Inventory.SKU = generateProductSKU()

		// Convert to map for validation
		productData, err := structToMap(product)
		s.Require().NoError(err, "Failed to convert struct to map")

		// Validate against schema
		result := schema.Validate(productData)
		s.True(result.Valid, "Product %s should be valid", product.ID)
	}

	// Test invalid data
	invalidProduct := map[string]interface{}{
		"id":    "INVALID-ID", // Wrong format
		"name":  "A",          // Too short
		"price": float64(-10), // Negative price
	}

	result := schema.Validate(invalidProduct)
	s.False(result.Valid, "Invalid product data should fail validation")
}

// Helper function to convert a struct to a map
func structToMap(obj interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Helper function to generate a random theme
func generateRandomTheme() string {
	themes := []string{"light", "dark", "system"}
	return themes[rand.Intn(len(themes))]
}

// Helper function to generate a random category
func generateRandomCategory() string {
	categories := []string{"electronics", "clothing", "books", "home", "beauty", "sports", "food", "other"}
	return categories[rand.Intn(len(categories))]
}

// Helper function to generate a product ID
func generateProductID() string {
	return fmt.Sprintf("PROD-%06d", rand.Intn(900000)+100000)
}

// Helper function to generate a product name
func generateProductName() string {
	adjectives := []string{"Premium", "Deluxe", "Advanced", "Smart", "Professional", "Ultra", "Compact"}
	nouns := []string{"Widget", "Device", "Gadget", "Tool", "System", "Product", "Solution"}

	return fmt.Sprintf("%s %s", adjectives[rand.Intn(len(adjectives))], nouns[rand.Intn(len(nouns))])
}

// Helper function to generate a product SKU
func generateProductSKU() string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	sku := "SKU-"
	for i := 0; i < 8; i++ {
		sku += string(chars[rand.Intn(len(chars))])
	}
	return sku
}
