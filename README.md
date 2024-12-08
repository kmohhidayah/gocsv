# gocsv

A flexible and efficient CSV reader for Go that supports automatic struct mapping and custom time formats.

## Features

- Automatic mapping of CSV columns to struct fields
- Support for custom struct tags
- Built-in type conversion for common Go types
- Flexible date/time format handling
- Pointer type support
- Clean and simple API

## Installation

```bash
go get github.com/kmohhidayah/gocsv
```

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "github.com/kmohhidayah/gocsv"
)

type Person struct {
    Name      string    `csv:"name"`
    Age       int       `csv:"age"`
    BirthDate time.Time `csv:"birth_date"`
}

func main() {
    // Create a new CSV reader
    reader, err := gocsv.NewCSVReader("people.csv")
    if err != nil {
        panic(err)
    }
    defer reader.Close()

    // Read records one by one
    var person Person
    for {
        err := reader.ReadNext(&person)
        if err != nil {
            if err == io.EOF {
                break
            }
            panic(err)
        }
        fmt.Printf("Name: %s, Age: %d, Birth Date: %s\n", 
            person.Name, person.Age, person.BirthDate)
    }
}
```

### Custom Time Layout

```go
reader, err := gocsv.NewCSVReader("data.csv")
if err != nil {
    panic(err)
}
// Set custom time layout for all date fields
reader.SetTimeLayout("02/01/2006")
```

### Using Custom Time Format Per Field

```go
type Record struct {
    CreatedAt time.Time `csv:"created_at,2006-01-02 15:04:05"`
    UpdatedAt time.Time `csv:"updated_at,02/01/2006"`
}
```

## Supported Types

- `string`
- `int`, `int8`, `int16`, `int32`, `int64`
- `float32`, `float64`
- `bool`
- `time.Time`
- Pointer versions of all above types

## Struct Tags

The package uses struct tags to map CSV columns to struct fields:

```go
type Example struct {
    // Use the same name as CSV header
    Name string `csv:"name"`
    
    // Custom time format
    Date time.Time `csv:"date,02/01/2006"`
    
    // Field without tag will use struct field name
    Age int
}
```

## Error Handling

The package provides detailed error messages for common issues:
- File opening errors
- CSV parsing errors
- Type conversion errors
- Invalid time format errors
- Missing or mismatched headers

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
