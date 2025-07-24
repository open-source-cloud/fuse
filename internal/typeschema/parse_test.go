package typeschema_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/open-source-cloud/fuse/internal/typeschema"
)

type ParseValueTestSuite struct {
	suite.Suite
}

func TestParseValueTestSuite(t *testing.T) {
	suite.Run(t, new(ParseValueTestSuite))
}

func (s *ParseValueTestSuite) TestStringParsing() {
	tests := []struct {
		name     string
		typeStr  string
		val      any
		expected any
		wantErr  bool
	}{
		{
			name:     "string from string",
			typeStr:  "string",
			val:      "hello",
			expected: "hello",
			wantErr:  false,
		},
		{
			name:     "string from int",
			typeStr:  "string",
			val:      42,
			expected: "42",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := typeschema.ParseValue(tt.typeStr, tt.val)

			if tt.wantErr {
				s.Error(err)
				return
			}

			s.NoError(err)
			s.Equal(tt.expected, got)
		})
	}
}

func (s *ParseValueTestSuite) TestIntParsing() {
	tests := []struct {
		name     string
		typeStr  string
		val      any
		expected any
		wantErr  bool
	}{
		{
			name:     "int from int",
			typeStr:  "int",
			val:      42,
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "int from int32",
			typeStr:  "int",
			val:      int32(42),
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "int from int64",
			typeStr:  "int",
			val:      int64(42),
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "int from float64",
			typeStr:  "int",
			val:      float64(42.0),
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "int from string",
			typeStr:  "int",
			val:      "42",
			expected: 42,
			wantErr:  false,
		},
		{
			name:    "int from invalid string",
			typeStr: "int",
			val:     "not a number",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := typeschema.ParseValue(tt.typeStr, tt.val)

			if tt.wantErr {
				s.Error(err)
				return
			}

			s.NoError(err)
			s.Equal(tt.expected, got)
		})
	}
}

func (s *ParseValueTestSuite) TestFloat64Parsing() {
	tests := []struct {
		name     string
		typeStr  string
		val      any
		expected any
		wantErr  bool
	}{
		{
			name:     "float64 from float64",
			typeStr:  "float64",
			val:      float64(42.5),
			expected: float64(42.5),
			wantErr:  false,
		},
		{
			name:     "float64 from int",
			typeStr:  "float64",
			val:      42,
			expected: float64(42),
			wantErr:  false,
		},
		{
			name:     "float64 from string",
			typeStr:  "float64",
			val:      "42.5",
			expected: float64(42.5),
			wantErr:  false,
		},
		{
			name:    "float64 from invalid string",
			typeStr: "float64",
			val:     "not a number",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := typeschema.ParseValue(tt.typeStr, tt.val)

			if tt.wantErr {
				s.Error(err)
				return
			}

			s.NoError(err)
			s.Equal(tt.expected, got)
		})
	}
}

func (s *ParseValueTestSuite) TestBoolParsing() {
	tests := []struct {
		name     string
		typeStr  string
		val      any
		expected any
		wantErr  bool
	}{
		{
			name:     "bool from bool",
			typeStr:  "bool",
			val:      true,
			expected: true,
			wantErr:  false,
		},
		{
			name:     "bool from string true",
			typeStr:  "bool",
			val:      "true",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "bool from string false",
			typeStr:  "bool",
			val:      "false",
			expected: false,
			wantErr:  false,
		},
		{
			name:    "bool from invalid string",
			typeStr: "bool",
			val:     "not a boolean",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := typeschema.ParseValue(tt.typeStr, tt.val)

			if tt.wantErr {
				s.Error(err)
				return
			}

			s.NoError(err)
			s.Equal(tt.expected, got)
		})
	}
}

func (s *ParseValueTestSuite) TestSliceParsing() {
	tests := []struct {
		name     string
		typeStr  string
		val      any
		expected any
		wantErr  bool
	}{
		{
			name:     "string slice from slice",
			typeStr:  "[]string",
			val:      []any{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
			wantErr:  false,
		},
		{
			name:     "int slice from slice",
			typeStr:  "[]int",
			val:      []any{1, 2, 3},
			expected: []int{1, 2, 3},
			wantErr:  false,
		},
		{
			name:     "float64 slice from slice",
			typeStr:  "[]float64",
			val:      []any{1.1, 2.2, 3.3},
			expected: []float64{1.1, 2.2, 3.3},
			wantErr:  false,
		},
		{
			name:    "invalid slice element",
			typeStr: "[]int",
			val:     []any{1, "not a number", 3},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := typeschema.ParseValue(tt.typeStr, tt.val)

			if tt.wantErr {
				s.Error(err)
				return
			}

			s.NoError(err)
			s.Equal(tt.expected, got)
		})
	}
}

func (s *ParseValueTestSuite) TestUnsupportedTypes() {
	tests := []struct {
		name     string
		typeStr  string
		val      any
		expected any
		wantErr  bool
	}{
		{
			name:    "unsupported type",
			typeStr: "complex128",
			val:     complex128(1 + 2i),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := typeschema.ParseValue(tt.typeStr, tt.val)

			if tt.wantErr {
				s.Error(err)
				return
			}

			s.NoError(err)
			s.Equal(tt.expected, got)
		})
	}
}

func (s *ParseValueTestSuite) TestMapByteToMapStringAny() {
	tests := []struct {
		name     string
		typeStr  string
		val      any
		expected any
		wantErr  bool
	}{
		{
			name:     "map[string]any from []byte",
			typeStr:  "map[string]any",
			val:      []byte(`{"a": 1, "b": 2}`),
			expected: map[string]any{"a": float64(1), "b": float64(2)},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := typeschema.ParseValue(tt.typeStr, tt.val)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.Equal(tt.expected, got)
		})
	}
}
