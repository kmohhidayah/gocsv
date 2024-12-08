package gocsv

import (
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type CSVReader struct {
	reader     *csv.Reader
	file       *os.File
	headers    []string
	timeLayout string // Menambahkan layout waktu default
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
		reader:     reader,
		file:       file,
		headers:    headers,
		timeLayout: "2006-01-02", // Layout default untuk parsing tanggal
	}, nil
}

// SetTimeLayout memungkinkan pengguna mengatur format waktu kustom
func (r *CSVReader) SetTimeLayout(layout string) {
	r.timeLayout = layout
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

	for i, header := range r.headers {
		headerMap[header] = i
	}

	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		fieldValue := destValue.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		csvTag := field.Tag.Get("csv")
		if csvTag == "" {
			csvTag = field.Name
		}

		// Parse opsi tambahan dari tag
		tagOptions := strings.Split(csvTag, ",")
		csvTag = tagOptions[0]
		timeFormat := r.timeLayout
		if len(tagOptions) > 1 {
			timeFormat = tagOptions[1] // Menggunakan format waktu kustom jika ada
		}

		columnIndex, ok := headerMap[csvTag]
		if !ok {
			continue
		}

		value := strings.TrimSpace(record[columnIndex])
		if value == "" {
			continue
		}

		switch fieldValue.Kind() {
		case reflect.Ptr:
			if fieldValue.IsNil() {
				fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
			}
			err := r.setFieldValue(fieldValue.Elem(), value, timeFormat)
			if err != nil {
				return fmt.Errorf("error setting pointer value for %s: %w", csvTag, err)
			}
		default:
			err := r.setFieldValue(fieldValue, value, timeFormat)
			if err != nil {
				return fmt.Errorf("error setting value for %s: %w", csvTag, err)
			}
		}
	}
	return nil
}

func (r *CSVReader) setFieldValue(fieldValue reflect.Value, value string, timeFormat string) error {
	// Cek apakah field adalah time.Time
	if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
		t, err := time.Parse(timeFormat, value)
		if err != nil {
			return fmt.Errorf("error parsing time: %w", err)
		}
		fieldValue.Set(reflect.ValueOf(t))
		return nil
	}

	// Handle tipe data lainnya
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
