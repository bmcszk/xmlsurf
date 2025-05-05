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
- Memory-efficient with optimized string operations
- Modular, well-structured codebase for maintainability

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
    fmt.Println(result["/root/item[1]"])     // Output: first
    fmt.Println(result["/root/item[2]"])     // Output: second
    fmt.Println(result["/root/item[1]/@id"]) // Output: 1
    fmt.Println(result["/root/item[2]/@id"]) // Output: 2
    
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

// Get detailed differences between XMLMaps
diffs := map1.Diffs(map2)
for _, diff := range diffs {
    fmt.Println(diff.String()) // Human-readable description of the difference
}

// Get detailed differences ignoring element order
diffs := map1.DiffsIgnoreOrder(map2)
```

The `Diff` struct provides detailed information about differences between XML maps:

```go
type Diff struct {
    Path       string   // The XPath where the difference was found
    LeftValue  string   // Value in the left XMLMap (empty if path doesn't exist)
    RightValue string   // Value in the right XMLMap (empty if path doesn't exist)
    Type       DiffType // Type of difference (DiffMissing, DiffExtra, or DiffValue)
}
```

Diff types:
- `DiffMissing` - Path exists in right but not in left
- `DiffExtra` - Path exists in left but not in right
- `DiffValue` - Path exists in both but values differ

## Path Representation

The XMLMap uses XPath-like path expressions as keys:

- Basic element paths: `/root/child`
- Element indices for repeated elements: `/root/items/item[1]`, `/root/items/item[2]`
- Attribute paths: `/root/element/@attribute`
- Namespaced elements: `/ns:root/ns:child`

## Implementation Details

The library has been optimized for performance and memory efficiency:

- Uses a string builder pool to minimize memory allocations
- Pre-allocates collections with appropriate initial sizes
- Efficiently handles element repetition with automatic indexing
- Optimized string operations to reduce concatenation overhead
- Modular, well-organized code structure for maintainability

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
