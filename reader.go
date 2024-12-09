package gocsv

import (
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CSVReader struct {
	reader     *csv.Reader
	file       *os.File
	headers    []string
	headerMap  map[string]int
	timeLayout string
	mu         sync.RWMutex
}

// NewCSVReader creates a new CSV reader with the specified file path
func NewCSVReader(filePath string) (*CSVReader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, &CSVError{Field: "file", Value: filePath, Wrapped: err}
	}

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		file.Close()
		return nil, &CSVError{Field: "headers", Wrapped: err}
	}

	// Initialize header map
	headerMap := make(map[string]int, len(headers))
	for i, header := range headers {
		headerMap[header] = i
	}

	return &CSVReader{
		reader:     reader,
		file:       file,
		headers:    headers,
		headerMap:  headerMap,
		timeLayout: DateOnly, // Default layout
	}, nil
}

func (r *CSVReader) SetTimeLayout(layout string) error {
	if err := r.ValidateTimeLayout(layout); err != nil {
		return &CSVError{
			Field:   "timeLayout",
			Value:   layout,
			Type:    "string",
			Wrapped: err,
		}
	}
	r.mu.Lock()
	r.timeLayout = layout
	r.mu.Unlock()
	return nil
}

// ValidateTimeLayout validates the time layout format
func (r *CSVReader) ValidateTimeLayout(layout string) error {
	if layout == "" {
		return fmt.Errorf("time layout cannot be empty")
	}

	// Verify that layout contains at least year, month, and day components
	hasYear := strings.Contains(layout, "2006")
	hasMonth := strings.Contains(layout, "01") || strings.Contains(layout, "Jan")
	hasDay := strings.Contains(layout, "02")

	if !hasYear || !hasMonth || !hasDay {
		return fmt.Errorf("invalid time layout: must contain at least year, month, and day components")
	}

	// Reference time used by Go for time formatting
	referenceTime := time.Date(2006, time.January, 02, 15, 04, 05, 0, time.UTC)
	formatted := referenceTime.Format(layout)

	// Try to parse the formatted date using the provided layout
	parsedTime, err := time.Parse(layout, formatted)
	if err != nil {
		return fmt.Errorf("invalid time layout %s: %v", layout, err)
	}

	// Additional validation: ensure the parsed time matches the reference time
	// This helps catch cases where the layout might parse successfully but lose information
	expectedFormatted := parsedTime.Format(layout)
	if formatted != expectedFormatted {
		return fmt.Errorf("invalid time layout: inconsistent parsing results")
	}

	return nil
}

// ReadNext reads the next record and populates the provided struct
func (r *CSVReader) ReadNext(dest interface{}) error {
	record, err := r.reader.Read()
	if err != nil {
		return err
	}

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.IsNil() {
		return &CSVError{Field: "destination", Type: "pointer",
			Value: fmt.Sprintf("%T", dest)}
	}

	destValue = destValue.Elem()
	if destValue.Kind() != reflect.Struct {
		return &CSVError{Field: "destination", Type: "struct",
			Value: fmt.Sprintf("%T", dest)}
	}

	return r.populateStruct(destValue, record)
}

func (r *CSVReader) populateStruct(destValue reflect.Value, record []string) error {
	destType := destValue.Type()

	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		fieldValue := destValue.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		tag := r.parseCSVTag(field)
		if tag.name == "-" {
			continue
		}

		columnIndex, ok := r.headerMap[tag.name]
		if !ok {
			continue
		}

		if columnIndex >= len(record) {
			return &CSVError{Field: tag.name, Value: "index out of range"}
		}

		value := strings.TrimSpace(record[columnIndex])
		if value == "" {
			continue
		}

		if err := r.setFieldValue(fieldValue, value, tag.timeFormat, field.Name); err != nil {
			return err
		}
	}

	return nil
}

type csvTag struct {
	name       string
	timeFormat string
}

func (r *CSVReader) parseCSVTag(field reflect.StructField) csvTag {
	tag := field.Tag.Get("csv")
	if tag == "" {
		return csvTag{name: field.Name, timeFormat: r.timeLayout}
	}

	parts := strings.Split(tag, ",")
	if len(parts) == 1 {
		return csvTag{name: parts[0], timeFormat: r.timeLayout}
	}

	return csvTag{name: parts[0], timeFormat: parts[1]}
}

func (r *CSVReader) setFieldValue(fieldValue reflect.Value, value string, timeFormat, fieldName string) error {
	fieldNameLower := strings.ToLower(fieldName)

	// Handle pointer types
	if fieldValue.Kind() == reflect.Ptr {
		if fieldValue.IsNil() {
			fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
		}
		return r.setFieldValue(fieldValue.Elem(), value, timeFormat, fieldName)
	}

	// Handle time.Time
	if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
		return r.setTimeValue(fieldValue, value, timeFormat, fieldNameLower)
	}

	// Handle basic types
	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(value)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return &CSVError{
				Field:   fieldNameLower,
				Value:   value,
				Type:    "int",
				Wrapped: err,
			}
		}
		fieldValue.SetInt(intVal)
		return nil

	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return &CSVError{
				Field:   fieldNameLower,
				Value:   value,
				Type:    "float",
				Wrapped: err,
			}
		}
		fieldValue.SetFloat(floatVal)
		return nil

	case reflect.Bool:
		boolVal, err := parseBool(value)
		if err != nil {
			return &CSVError{
				Field:   fieldNameLower,
				Value:   value,
				Type:    "bool",
				Wrapped: err,
			}
		}
		fieldValue.SetBool(boolVal)
		return nil

	default:
		return &CSVError{
			Field: fieldNameLower,
			Value: value,
			Type:  fieldValue.Kind().String(),
		}
	}
}

func (r *CSVReader) setTimeValue(fieldValue reflect.Value, value, timeFormat, fieldName string) error {
	t, err := time.Parse(timeFormat, value)
	if err != nil {
		// Coba parse dengan format default jika format custom gagal
		sanitizedValue, sanitizeErr := r.sanitizeTimeValue(value)
		if sanitizeErr != nil {
			return &CSVError{
				Field:   fieldName,
				Value:   value,
				Type:    "time.Time",
				Wrapped: err,
			}
		}
		t, err = time.Parse(r.timeLayout, sanitizedValue)
		if err != nil {
			return &CSVError{
				Field:   fieldName,
				Value:   value,
				Type:    "time.Time",
				Wrapped: err,
			}
		}
	}
	fieldValue.Set(reflect.ValueOf(t))
	return nil
}

// Tambahkan helper function untuk parsing boolean
func parseBool(value string) (bool, error) {
	value = strings.ToLower(value)
	switch value {
	case "true", "1", "yes", "y":
		return true, nil
	case "false", "0", "no", "n":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", value)
	}
}

func (r *CSVReader) sanitizeTimeValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	commonLayouts := []string{
		Layout, ANSIC, UnixDate, RubyDate, RFC822, RFC822Z,
		RFC850, RFC1123, RFC1123Z, RFC3339, RFC3339Nano,
		Kitchen, Stamp, StampMilli, StampMicro, StampNano,
		DateTime, DateOnly, TimeOnly,
	}

	for _, layout := range commonLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.Format(r.timeLayout), nil
		}
	}

	return "", fmt.Errorf("unable to parse time value: %s", value)
}

// Close closes the underlying file
func (r *CSVReader) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}
