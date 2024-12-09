package gocsv

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// TestStruct represents a test structure with various field types
type TestStruct struct {
	StringField  string    `csv:"string_field"`
	IntField     int       `csv:"int_field"`
	FloatField   float64   `csv:"float_field"`
	BoolField    bool      `csv:"bool_field"`
	DateField    time.Time `csv:"date_field"`
	OptionalPtr  *string   `csv:"optional_field"`
	IgnoredField string    `csv:"-"`
}

func TestNewCSVReader(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name: "valid csv",
			content: `string_field,int_field,float_field,bool_field,date_field,optional_field
value1,123,45.67,true,2024-01-01,optional`,
			expectError: false,
		},
		{
			name:        "empty file",
			content:     "",
			expectError: true,
		},
		{
			name:        "invalid file path",
			content:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile := createTempFile(t, tt.content)
			defer os.Remove(tmpFile)

			reader, err := NewCSVReader(tmpFile)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if reader == nil {
				t.Error("expected reader to not be nil")
				return
			}

			defer reader.Close()
		})
	}
}

func TestReadNext(t *testing.T) {
	content := `string_field,int_field,float_field,bool_field,date_field,optional_field
value1,123,45.67,true,2024-01-01,optional
value2,-456,78.90,false,2024-02-01,
value3,789,12.34,yes,2024-03-01,test`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	reader, err := NewCSVReader(tmpFile)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	defer reader.Close()

	expected := []TestStruct{
		{
			StringField: "value1",
			IntField:    123,
			FloatField:  45.67,
			BoolField:   true,
			DateField:   mustParseTime("2024-01-01"),
			OptionalPtr: strPtr("optional"),
		},
		{
			StringField: "value2",
			IntField:    -456,
			FloatField:  78.90,
			BoolField:   false,
			DateField:   mustParseTime("2024-02-01"),
			OptionalPtr: nil,
		},
		{
			StringField: "value3",
			IntField:    789,
			FloatField:  12.34,
			BoolField:   true,
			DateField:   mustParseTime("2024-03-01"),
			OptionalPtr: strPtr("test"),
		},
	}

	for i, exp := range expected {
		var got TestStruct
		err := reader.ReadNext(&got)
		if err != nil {
			t.Errorf("row %d: unexpected error: %v", i, err)
			continue
		}

		if got.StringField != exp.StringField {
			t.Errorf("row %d: StringField: got %v, want %v", i, got.StringField, exp.StringField)
		}
		if got.IntField != exp.IntField {
			t.Errorf("row %d: IntField: got %v, want %v", i, got.IntField, exp.IntField)
		}
		if got.FloatField != exp.FloatField {
			t.Errorf("row %d: FloatField: got %v, want %v", i, got.FloatField, exp.FloatField)
		}
		if got.BoolField != exp.BoolField {
			t.Errorf("row %d: BoolField: got %v, want %v", i, got.BoolField, exp.BoolField)
		}
		if !got.DateField.Equal(exp.DateField) {
			t.Errorf("row %d: DateField: got %v, want %v", i, got.DateField, exp.DateField)
		}
		if (got.OptionalPtr == nil) != (exp.OptionalPtr == nil) {
			t.Errorf("row %d: OptionalPtr: got %v, want %v", i, got.OptionalPtr, exp.OptionalPtr)
		} else if got.OptionalPtr != nil && *got.OptionalPtr != *exp.OptionalPtr {
			t.Errorf("row %d: OptionalPtr value: got %v, want %v", i, *got.OptionalPtr, *exp.OptionalPtr)
		}
	}
}

func TestSetTimeLayout(t *testing.T) {
	tests := []struct {
		name        string
		layout      string
		expectError bool
	}{
		{
			name:        "valid layout - standard",
			layout:      "2006-01-02",
			expectError: false,
		},
		{
			name:        "valid layout - with time",
			layout:      "2006-01-02 15:04:05",
			expectError: false,
		},
		{
			name:        "empty layout",
			layout:      "",
			expectError: true,
		},
		{
			name:        "invalid layout - random string",
			layout:      "invalid-format",
			expectError: true,
		},
		{
			name:        "invalid layout - partial date",
			layout:      "2006",
			expectError: true,
		},
		{
			name:        "invalid layout - wrong reference date",
			layout:      "2007-01-02",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &CSVReader{
				timeLayout: "2006-01-02", // default layout
			}
			err := reader.SetTimeLayout(tt.layout)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for layout '%s', got nil", tt.layout)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for layout '%s': %v", tt.layout, err)
				}
			}
		})
	}
}

// Helper functions
func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	return tmpFile
}

func mustParseTime(value string) time.Time {
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		panic(err)
	}
	return t
}

func strPtr(s string) *string {
	return &s
}

// BenchStruct represents a test structure for benchmarking
type BenchStruct struct {
	StringField string    `csv:"string_field"`
	IntField    int       `csv:"int_field"`
	FloatField  float64   `csv:"float_field"`
	BoolField   bool      `csv:"bool_field"`
	DateField   time.Time `csv:"date_field"`
	OptionalPtr *string   `csv:"optional_field"`
}

// generateCSVContent generates CSV content with the specified number of rows
func generateCSVContent(rows int) string {
	content := "string_field,int_field,float_field,bool_field,date_field,optional_field\n"
	for i := 0; i < rows; i++ {
		row := fmt.Sprintf("value%d,%d,%f,%t,2024-01-%02d,optional%d\n",
			i, i, float64(i)*1.5, i%2 == 0, (i%28)+1, i)
		content += row
	}
	return content
}

// setupBenchmarkFile creates a temporary CSV file with the specified number of rows
func setupBenchmarkFile(b *testing.B, rows int) (string, func()) {
	b.Helper()
	content := generateCSVContent(rows)

	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "bench.csv")

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		b.Fatalf("failed to create benchmark file: %v", err)
	}

	cleanup := func() {
		os.Remove(tmpFile)
	}

	return tmpFile, cleanup
}

// BenchmarkNewCSVReader benchmarks the creation of new CSV readers
func BenchmarkNewCSVReader(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			fileName, cleanup := setupBenchmarkFile(b, size)
			defer cleanup()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				reader, err := NewCSVReader(fileName)
				if err != nil {
					b.Fatalf("failed to create reader: %v", err)
				}
				reader.Close()
			}
		})
	}
}

// BenchmarkReadNext benchmarks reading records with different file sizes
func BenchmarkReadNext(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			fileName, cleanup := setupBenchmarkFile(b, size)
			defer cleanup()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				reader, err := NewCSVReader(fileName)
				if err != nil {
					b.Fatalf("failed to create reader: %v", err)
				}

				b.StartTimer()
				var dest BenchStruct
				for j := 0; j < size; j++ {
					if err := reader.ReadNext(&dest); err != nil {
						b.Fatalf("failed to read record: %v", err)
					}
				}

				b.StopTimer()
				reader.Close()
			}
		})
	}
}

// BenchmarkReadNextParallel benchmarks parallel reading from multiple goroutines
func BenchmarkReadNextParallel(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			fileName, cleanup := setupBenchmarkFile(b, size)
			defer cleanup()

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					reader, err := NewCSVReader(fileName)
					if err != nil {
						b.Fatalf("failed to create reader: %v", err)
					}

					var dest BenchStruct
					for {
						if err := reader.ReadNext(&dest); err != nil {
							break
						}
					}

					reader.Close()
				}
			})
		})
	}
}

// BenchmarkSetTimeLayout benchmarks setting time layout with different formats
func BenchmarkSetTimeLayout(b *testing.B) {
	layouts := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"02/01/2006",
		"02-Jan-2006",
	}
	reader := &CSVReader{}
	for _, layout := range layouts {
		b.Run(fmt.Sprintf("layout_%s", layout), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := reader.SetTimeLayout(layout)
				if err != nil {
					b.Fatalf("failed to set time layout: %v", err)
				}
			}
		})
	}
}

// BenchmarkPopulateStruct benchmarks struct population with different field types
func BenchmarkPopulateStruct(b *testing.B) {
	record := []string{"test_string", "123", "45.67", "true", "2024-01-01", "optional"}
	reader := &CSVReader{
		headerMap: map[string]int{
			"string_field":   0,
			"int_field":      1,
			"float_field":    2,
			"bool_field":     3,
			"date_field":     4,
			"optional_field": 5,
		},
		timeLayout: "2006-01-02",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var dest BenchStruct
		err := reader.populateStruct(reflect.ValueOf(&dest).Elem(), record)
		if err != nil {
			b.Fatalf("failed to populate struct: %v", err)
		}
	}
}
