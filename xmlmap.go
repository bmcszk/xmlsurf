package xmlsurf

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// XMLMap represents a map of XPath expressions to their values
type XMLMap map[string]string

// Diff represents a difference between two XMLMaps
type Diff struct {
	Path       string   // The XPath where the difference was found
	LeftValue  string   // Value in the left XMLMap (empty if path doesn't exist)
	RightValue string   // Value in the right XMLMap (empty if path doesn't exist)
	Type       DiffType // Type of difference
}

// DiffType indicates the type of difference between XMLMaps
type DiffType int

const (
	// DiffMissing indicates a path exists in right but not in left
	DiffMissing DiffType = iota
	// DiffExtra indicates a path exists in left but not in right
	DiffExtra
	// DiffValue indicates a path exists in both but values differ
	DiffValue
)

// String returns a human-readable description of the difference
func (d Diff) String() string {
	switch d.Type {
	case DiffMissing:
		return fmt.Sprintf("Missing path: %s (right value: %q)", d.Path, d.RightValue)
	case DiffExtra:
		return fmt.Sprintf("Extra path: %s (left value: %q)", d.Path, d.LeftValue)
	case DiffValue:
		return fmt.Sprintf("Value mismatch at %s: %q != %q", d.Path, d.LeftValue, d.RightValue)
	default:
		return fmt.Sprintf("Unknown diff type at %s", d.Path)
	}
}

// ToXML converts the XMLMap to XML and writes it to the provided writer.
// The XML will be indented if indent is true.
func (m XMLMap) ToXML(w io.Writer, indent bool) error {
	if len(m) == 0 {
		return errors.New("empty XMLMap")
	}

	// Find the root element
	var rootPath string
	for path := range m {
		parts := strings.Split(path, "/")
		if len(parts) > 1 {
			rootPath = "/" + parts[1]
			break
		}
	}
	if rootPath == "" {
		return errors.New("no root element found")
	}

	// Build XML tree from map
	root, _, err := buildXMLTree(m, rootPath)
	if err != nil {
		return err
	}

	// Write XML
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	if indent {
		enc.Indent("", "  ")
	}

	// Write the root node and all its children
	if err := writeXMLNode(root, enc, comparePaths); err != nil {
		return err
	}

	if err := enc.Flush(); err != nil {
		return err
	}

	// Copy the buffer to the writer, skipping the XML header
	output := buf.String()
	if strings.HasPrefix(output, "<?xml") {
		if idx := strings.Index(output, "?>"); idx != -1 {
			output = output[idx+2:]
		}
	}
	_, err = io.WriteString(w, strings.TrimSpace(output))
	return err
}

// Equal returns true if two XMLMaps are equal
func (m XMLMap) Equal(other XMLMap) bool {
	diffs := m.findDiffs(other)
	return len(diffs) == 0
}

// Diffs returns a list of differences between two XMLMaps
// It compares exact paths and values, considering element order
func (m XMLMap) Diffs(other XMLMap) []Diff {
	return m.findDiffs(other)
}

// findDiffs is a helper method that finds differences between two XMLMaps
// It is used by both Equal and Diffs to avoid code duplication
func (m XMLMap) findDiffs(other XMLMap) []Diff {
	diffs := make([]Diff, 0)

	// Quick size check
	if len(m) != len(other) {
		// Maps have different sizes - find specific differences

		// Find paths in m that are missing or have different values in other
		for path, value := range m {
			otherValue, exists := other[path]
			if !exists {
				diffs = append(diffs, Diff{
					Path:      path,
					LeftValue: value,
					Type:      DiffExtra,
				})
			} else if value != otherValue {
				diffs = append(diffs, Diff{
					Path:       path,
					LeftValue:  value,
					RightValue: otherValue,
					Type:       DiffValue,
				})
			}
		}

		// Find paths in other that are missing in m
		for path, value := range other {
			if _, exists := m[path]; !exists {
				diffs = append(diffs, Diff{
					Path:       path,
					RightValue: value,
					Type:       DiffMissing,
				})
			}
		}
	} else {
		// Maps have same size - just check for differing values
		for path, value := range m {
			otherValue, exists := other[path]
			if !exists {
				diffs = append(diffs, Diff{
					Path:      path,
					LeftValue: value,
					Type:      DiffExtra,
				})
			} else if value != otherValue {
				diffs = append(diffs, Diff{
					Path:       path,
					LeftValue:  value,
					RightValue: otherValue,
					Type:       DiffValue,
				})
			}
		}
	}

	// Sort diffs by path for consistent output
	if len(diffs) > 0 {
		sort.Slice(diffs, func(i, j int) bool {
			return diffs[i].Path < diffs[j].Path
		})
	}

	return diffs
}

// EqualIgnoreOrder returns true if two XMLMaps are equal ignoring the order of elements
func (m XMLMap) EqualIgnoreOrder(other XMLMap) bool {
	diffs := m.findDiffsIgnoreOrder(other)
	return len(diffs) == 0
}

// DiffsIgnoreOrder returns a list of differences between two XMLMaps, ignoring element order
func (m XMLMap) DiffsIgnoreOrder(other XMLMap) []Diff {
	return m.findDiffsIgnoreOrder(other)
}

// findDiffsIgnoreOrder is a helper method that finds differences between two XMLMaps ignoring element order
// It is used by both EqualIgnoreOrder and DiffsIgnoreOrder to avoid code duplication
func (m XMLMap) findDiffsIgnoreOrder(other XMLMap) []Diff {
	diffs := make([]Diff, 0)

	// Create maps of values for each path
	values1 := make(map[string]map[string]bool, len(m)/2)
	values2 := make(map[string]map[string]bool, len(m)/2)

	// Maps to keep track of original paths before normalization
	pathsMap1 := make(map[string][]string)
	pathsMap2 := make(map[string][]string)

	// Reuse path builder to reduce allocations
	pathBuilder := getPathBuilder()
	defer putPathBuilder(pathBuilder)

	// Process the first map
	for k, v := range m {
		// Extract base path (removing indices)
		basePath := extractBasePath(k, pathBuilder)

		// Track the original paths for this base path
		pathsMap1[basePath] = append(pathsMap1[basePath], k)

		// Create value map if it doesn't exist
		if values1[basePath] == nil {
			values1[basePath] = make(map[string]bool)
		}
		values1[basePath][v] = true
	}

	// Process the second map
	for k, v := range other {
		// Extract base path (removing indices)
		basePath := extractBasePath(k, pathBuilder)

		// Track the original paths for this base path
		pathsMap2[basePath] = append(pathsMap2[basePath], k)

		// Create value map if it doesn't exist
		if values2[basePath] == nil {
			values2[basePath] = make(map[string]bool)
		}
		values2[basePath][v] = true
	}

	// Quick size check for optimization
	if len(values1) != len(values2) {
		// Different number of base paths - report all differences
		// Find missing paths and value differences
		collectDiffsFromValueSets(values1, values2, pathsMap1, pathsMap2, m, other, &diffs)
	} else {
		// Same number of base paths - check for value differences
		for basePath, vals1 := range values1 {
			vals2, exists := values2[basePath]
			if !exists || !mapSetsEqual(vals1, vals2) {
				// Either missing path or different values
				collectDiffsForBasePath(basePath, vals1, vals2, exists,
					pathsMap1, pathsMap2, m, other, &diffs)
			}
		}
	}

	// Sort diffs by path for consistent output
	if len(diffs) > 0 {
		sort.Slice(diffs, func(i, j int) bool {
			return diffs[i].Path < diffs[j].Path
		})
	}

	return diffs
}

// mapSetsEqual checks if two maps containing sets of values are equal
func mapSetsEqual(set1, set2 map[string]bool) bool {
	if len(set1) != len(set2) {
		return false
	}

	for v := range set1 {
		if !set2[v] {
			return false
		}
	}

	return true
}

// collectDiffsFromValueSets collects all differences between two value sets
// This is used when the number of base paths differs
func collectDiffsFromValueSets(
	values1, values2 map[string]map[string]bool,
	pathsMap1, pathsMap2 map[string][]string,
	m, other XMLMap,
	diffs *[]Diff) {

	// Find paths in values1 that are missing or have different values in values2
	for basePath, vals1 := range values1 {
		vals2, exists := values2[basePath]
		if !exists {
			// Base path missing from other
			for _, originalPath := range pathsMap1[basePath] {
				*diffs = append(*diffs, Diff{
					Path:      originalPath,
					LeftValue: m[originalPath],
					Type:      DiffExtra,
				})
			}
		} else {
			// Compare values - collect differences
			collectDiffsForBasePath(basePath, vals1, vals2, exists,
				pathsMap1, pathsMap2, m, other, diffs)
		}
	}

	// Find paths in values2 that are missing in values1
	for basePath := range values2 {
		if _, exists := values1[basePath]; !exists {
			// Base path missing from m
			for _, originalPath := range pathsMap2[basePath] {
				*diffs = append(*diffs, Diff{
					Path:       originalPath,
					RightValue: other[originalPath],
					Type:       DiffMissing,
				})
			}
		}
	}
}

// collectDiffsForBasePath collects all differences for a specific base path
func collectDiffsForBasePath(
	basePath string,
	vals1, vals2 map[string]bool,
	exists bool,
	pathsMap1, pathsMap2 map[string][]string,
	m, other XMLMap,
	diffs *[]Diff) {

	if !exists {
		// Path exists in left but not in right
		for _, originalPath := range pathsMap1[basePath] {
			*diffs = append(*diffs, Diff{
				Path:      originalPath,
				LeftValue: m[originalPath],
				Type:      DiffExtra,
			})
		}
		return
	}

	// Check for values in vals1 that don't exist in vals2
	for val := range vals1 {
		if !vals2[val] {
			// Find an original path with this value
			for _, path := range pathsMap1[basePath] {
				if m[path] == val {
					*diffs = append(*diffs, Diff{
						Path:      path,
						LeftValue: val,
						Type:      DiffExtra,
					})
					break
				}
			}
		}
	}

	// Check for values in vals2 that don't exist in vals1
	for val := range vals2 {
		if !vals1[val] {
			// Find an original path with this value
			for _, path := range pathsMap2[basePath] {
				if other[path] == val {
					*diffs = append(*diffs, Diff{
						Path:       path,
						RightValue: val,
						Type:       DiffMissing,
					})
					break
				}
			}
		}
	}
}
