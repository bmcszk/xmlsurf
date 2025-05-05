# xmlsurf

A modern and efficient Go library for XML processing that provides a simple way to work with XML data using XPath-style paths.

## Features

- Convert XML to and from a map structure using XPath-style paths
- Support for XML namespaces
- Attribute handling
- Value transformations
- Indented XML output
- Comprehensive error handling
- Order-independent comparison of XML structures

## Installation

```bash
go get github.com/bmcszk/xmlsurf
```

## Quick Start

```go
package main

import (
    "fmt"
    "strings"
    "github.com/bmcszk/xmlsurf"
)

func main() {
    // Parse XML to map
    xml := `<root>
        <item id="1">first</item>
        <item id="2">second</item>
    </root>`
    
    result, err := xmlsurf.ParseToMap(strings.NewReader(xml))
    if err != nil {
        panic(err)
    }
    
    // Access values using XPath-style paths
    fmt.Println(result["/root/item"])     // Output: [first second]
    fmt.Println(result["/root/item/@id"]) // Output: [1 2]
    
    // Convert back to XML
    var buf strings.Builder
    err = result.ToXML(&buf, true) // true for indented output
    if err != nil {
        panic(err)
    }
    fmt.Println(buf.String())
}
```

## Options

### Namespace Handling

```go
// Include namespace prefixes in element and attribute names
result, err := xmlsurf.ParseToMap(reader, xmlsurf.WithNamespaces(true))

// Exclude namespace prefixes
result, err := xmlsurf.ParseToMap(reader, xmlsurf.WithNamespaces(false))
```

### Value Transformations

```go
// Transform values during parsing
result, err := xmlsurf.ParseToMap(reader, 
    xmlsurf.WithValueTransform(strings.ToUpper),
    xmlsurf.WithValueTransform(strings.TrimSpace),
)
```

## Comparison Methods

```go
// Exact comparison (order matters)
equal := map1.Equal(map2)

// Order-independent comparison
equal := map1.EqualIgnoreOrder(map2)
```

## Error Handling

The library provides detailed error messages for various XML parsing scenarios:
- Invalid XML syntax
- Unclosed elements
- Multiple root elements
- Malformed attributes
- Invalid element names

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 
