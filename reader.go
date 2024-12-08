package gocsv

import (
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type CSVReader struct {
	reader  *csv.Reader
	file    *os.File
	headers []string
}

func NewCSVReader(filePath string) (*CSVReader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	reader := csv.NewReader(file)

	headers, err := reader.Read()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("error reading headers: %w", err)
	}

	return &CSVReader{
		reader:  reader,
		file:    file,
		headers: headers,
	}, nil
}

func (r *CSVReader) ReadNext(dest interface{}) error {
	record, err := r.reader.Read()
	if err != nil {
		return err
	}

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer to struct")
	}

	destValue = destValue.Elem()
	if destValue.Kind() != reflect.Struct {
		return fmt.Errorf("destination must be a struct")
	}

	destType := destValue.Type()
	headerMap := make(map[string]int)

	// Mapping header ke index
	for i, header := range r.headers {
		headerMap[header] = i
	}

	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		fieldValue := destValue.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		// Get nama kolom dari tag csv
		csvTag := field.Tag.Get("csv")
		if csvTag == "" {
			csvTag = field.Name
		}

		// Cari index dari header
		columnIndex, ok := headerMap[csvTag]
		if !ok {
			continue
		}

		// Get nilai dari CSV
		value := strings.TrimSpace(record[columnIndex])
		if value == "" {
			continue // Skip empty values
		}

		// Handle pointer dan non-pointer types
		switch fieldValue.Kind() {
		case reflect.Ptr:
			// Create instance baru untuk pointer jika nil
			if fieldValue.IsNil() {
				fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
			}
			// Set nilai ke pointer
			err := setFieldValue(fieldValue.Elem(), value)
			if err != nil {
				return fmt.Errorf("error setting pointer value for %s: %w", csvTag, err)
			}
		default:
			// Set nilai langsung untuk non-pointer
			err := setFieldValue(fieldValue, value)
			if err != nil {
				return fmt.Errorf("error setting value for %s: %w", csvTag, err)
			}
		}
	}

	return nil
}

// setFieldValue mengatur nilai field berdasarkan typenya
func setFieldValue(fieldValue reflect.Value, value string) error {
	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("error converting to int: %w", err)
		}
		fieldValue.SetInt(int64(intVal))

	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("error converting to float: %w", err)
		}
		fieldValue.SetFloat(floatVal)

	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("error converting to bool: %w", err)
		}
		fieldValue.SetBool(boolVal)

	default:
		return fmt.Errorf("unsupported field type: %v", fieldValue.Kind())
	}

	return nil
}

func (r *CSVReader) Close() error {
	return r.file.Close()
}
