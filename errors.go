package gocsv

import "fmt"

type CSVError struct {
	Field   string
	Value   string
	Type    string
	Wrapped error
}

func (e *CSVError) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("field %s: error converting value '%s' to %s: %v",
			e.Field, e.Value, e.Type, e.Wrapped)
	}
	return fmt.Sprintf("field %s: error with value '%s' of type %s",
		e.Field, e.Value, e.Type)
}
