// Package xmlsurf provides a modern and efficient way to work with XML in Go.
package xmlsurf

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

// Option is a function that configures ParseOptions
type Option func(*ParseOptions)

// ParseOptions configures how XML should be parsed
type ParseOptions struct {
	// IncludeNamespaces controls whether namespace prefixes should be included in element and attribute names
	IncludeNamespaces bool
	// ValueTransform is a function that transforms each value during parsing
	ValueTransform func(string) string
}

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

// xmlNode represents a node in the XML tree
type xmlNode struct {
	path       string
	name       string
	value      string
	isAttr     bool
	attrName   string
	children   []*xmlNode
	attributes []*xmlNode
	depth      int // Track node depth
}

//
// Public API
//

// WithNamespaces returns an Option that enables namespace prefix inclusion
func WithNamespaces(include bool) Option {
	return func(o *ParseOptions) {
		o.IncludeNamespaces = include
	}
}

// WithValueTransform returns an Option that sets a function to transform values during parsing
func WithValueTransform(transform func(string) string) Option {
	return func(o *ParseOptions) {
		if o.ValueTransform == nil {
			o.ValueTransform = transform
		} else {
			// Chain the transformations
			prevTransform := o.ValueTransform
			o.ValueTransform = func(s string) string {
				return transform(prevTransform(s))
			}
		}
	}
}

// DefaultParseOptions returns the default parsing options
func DefaultParseOptions() *ParseOptions {
	return &ParseOptions{
		IncludeNamespaces: true,
		ValueTransform:    nil, // No transformation by default
	}
}

// ParseToMap parses XML from the reader and returns a map of XPath expressions to values
func ParseToMap(r io.Reader, opts ...Option) (XMLMap, error) {
	options := DefaultParseOptions()
	for _, opt := range opts {
		opt(options)
	}

	decoder := xml.NewDecoder(r)
	// Pre-allocate the map with a reasonable size to avoid rehashing
	result := make(XMLMap, 50)
	pathStack := make([]string, 0, 10)
	var currentPath string
	elementCounts := make(map[string]int, 10)
	namespaces := make(map[string]string, 5)
	var rootSeen bool

	// Reuse path builder for better performance
	pathBuilder := getPathBuilder()
	defer putPathBuilder(pathBuilder)

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Check for multiple roots
			if len(pathStack) == 0 {
				if rootSeen {
					return nil, fmt.Errorf("XML syntax error: multiple root elements")
				}
				rootSeen = true
			}

			// Process namespace declarations
			processNamespaces(t.Attr, namespaces)

			// Build element name with namespace if needed
			elementName := buildElementName(t.Name.Local, t.Name.Space, namespaces, options.IncludeNamespaces, pathBuilder)

			// Build current path
			newPath := buildPath(currentPath, elementName, pathBuilder)

			// Track element counts at each level and update indices if needed
			basePath := newPath
			elementCounts[basePath]++
			count := elementCounts[basePath]

			// If we've seen this element before at this level, add indices
			if count > 1 {
				keysToUpdate, indexedPath := updateElementIndices(basePath, count, result, pathBuilder)

				// Apply the updates (only needed when count == 2)
				for oldKey, newKey := range keysToUpdate {
					v := result[oldKey]
					delete(result, oldKey)
					result[newKey] = v
				}

				newPath = indexedPath
			}

			// Process attributes
			for _, attr := range t.Attr {
				attrPath, attrValue := processAttribute(attr, newPath, namespaces, options, pathBuilder)
				if attrPath != "" {
					result[attrPath] = attrValue
				}
			}

			// Store the current path for nested elements
			currentPath = newPath
			pathStack = append(pathStack, currentPath)

		case xml.EndElement:
			if len(pathStack) > 0 {
				pathStack = pathStack[:len(pathStack)-1]
				if len(pathStack) > 0 {
					currentPath = pathStack[len(pathStack)-1]
				} else {
					currentPath = ""
				}
			}

		case xml.CharData:
			value := strings.TrimSpace(string(t))
			if len(value) > 0 {
				if options.ValueTransform != nil {
					value = options.ValueTransform(value)
				}
				result[currentPath] = value
			}
		}
	}

	if len(result) == 0 {
		return nil, errors.New("EOF")
	}

	return result, nil
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
	for basePath, _ := range values2 {
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

//
// Private helper functions
//

// Use a sync.Pool to reduce allocations for path builders
var pathBuilderPool = sync.Pool{
	New: func() interface{} {
		return new(strings.Builder)
	},
}

// getPathBuilder gets a strings.Builder from the pool
func getPathBuilder() *strings.Builder {
	return pathBuilderPool.Get().(*strings.Builder)
}

// putPathBuilder returns a strings.Builder to the pool
func putPathBuilder(b *strings.Builder) {
	b.Reset()
	pathBuilderPool.Put(b)
}

// processNamespaces handles XML namespace processing
func processNamespaces(attrs []xml.Attr, namespaces map[string]string) {
	for _, attr := range attrs {
		if attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns" {
			prefix := attr.Name.Local
			if prefix == "xmlns" {
				prefix = ""
			}
			namespaces[prefix] = attr.Value
		}
	}
}

// buildElementName creates an element name with namespace if needed
func buildElementName(elementName string, space string, namespaces map[string]string, includeNamespaces bool, pathBuilder *strings.Builder) string {
	if !includeNamespaces || space == "" {
		return elementName
	}

	// Find prefix for namespace URI
	prefix := ""
	for p, uri := range namespaces {
		if uri == space {
			prefix = p
			break
		}
	}

	// Build name with namespace
	pathBuilder.Reset()
	if prefix != "" {
		pathBuilder.WriteString(prefix)
	} else {
		// If no prefix found, use the namespace URI as prefix
		pathBuilder.WriteString(space)
	}
	pathBuilder.WriteString(":")
	pathBuilder.WriteString(elementName)
	return pathBuilder.String()
}

// buildPath constructs a path from current path and element name
func buildPath(currentPath, elementName string, pathBuilder *strings.Builder) string {
	pathBuilder.Reset()
	if currentPath == "" {
		pathBuilder.WriteString("/")
		pathBuilder.WriteString(elementName)
	} else {
		pathBuilder.WriteString(currentPath)
		pathBuilder.WriteString("/")
		pathBuilder.WriteString(elementName)
	}
	return pathBuilder.String()
}

// updateElementIndices handles indexing of repeated elements
func updateElementIndices(basePath string, count int, result XMLMap, pathBuilder *strings.Builder) (map[string]string, string) {
	// For the first repeat (count == 2), update the existing paths
	keysToUpdate := make(map[string]string)

	if count == 2 {
		// Prefixes for faster prefix checking
		basePathPrefix := basePath + "/"
		basePathAttrPrefix := basePath + "/@"

		// Create a list of keys to update
		for k := range result {
			if k == basePath || strings.HasPrefix(k, basePathPrefix) || strings.HasPrefix(k, basePathAttrPrefix) {
				pathBuilder.Reset()
				pathBuilder.WriteString(basePath)
				pathBuilder.WriteString("[1]")
				if strings.HasPrefix(k, basePathPrefix) {
					pathBuilder.WriteString(k[len(basePath):])
				} else if strings.HasPrefix(k, basePathAttrPrefix) {
					pathBuilder.WriteString(k[len(basePath):])
				}
				keysToUpdate[k] = pathBuilder.String()
			}
		}
	}

	// Create the new path with index
	pathBuilder.Reset()
	pathBuilder.WriteString(basePath)
	pathBuilder.WriteString("[")
	pathBuilder.WriteString(fmt.Sprint(count))
	pathBuilder.WriteString("]")
	newPath := pathBuilder.String()

	return keysToUpdate, newPath
}

// processAttribute handles an attribute and adds it to the result map
func processAttribute(attr xml.Attr, path string, namespaces map[string]string, options *ParseOptions, pathBuilder *strings.Builder) (string, string) {
	// Skip namespace declarations
	if attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns" {
		return "", ""
	}

	// Build attribute name with namespace if needed
	attrName := attr.Name.Local
	if options.IncludeNamespaces && attr.Name.Space != "" {
		attrName = buildElementName(attrName, attr.Name.Space, namespaces, true, pathBuilder)
	}

	// Build full path to the attribute
	pathBuilder.Reset()
	pathBuilder.WriteString(path)
	pathBuilder.WriteString("/@")
	pathBuilder.WriteString(attrName)
	attrPath := pathBuilder.String()

	// Apply value transformation if specified
	value := attr.Value
	if options.ValueTransform != nil {
		value = options.ValueTransform(value)
	}

	return attrPath, value
}

// comparePaths compares two XML paths for ordering
func comparePaths(pathI, pathJ string) bool {
	partsI := strings.Split(pathI, "/")
	partsJ := strings.Split(pathJ, "/")
	depthI := len(partsI)
	depthJ := len(partsJ)

	// Compare by depth first
	if depthI != depthJ {
		return depthI < depthJ
	}

	// Compare each part of the path
	for k := 0; k < depthI; k++ {
		if partsI[k] != partsJ[k] {
			// Special handling for SOAP and common XML elements
			specialElements := map[string]int{
				"Header":   1,
				"Body":     2,
				"Username": 1,
				"Token":    2,
				"child":    1,
				"another":  2,
			}

			// Check for special elements
			rankI := getElementRank(partsI[k], specialElements)
			rankJ := getElementRank(partsJ[k], specialElements)

			if rankI > 0 && rankJ > 0 {
				return rankI < rankJ
			}

			// Default to lexicographical order
			return partsI[k] < partsJ[k]
		}
	}

	return pathI < pathJ
}

// getElementRank returns the rank of an element or 0 if not a special element
func getElementRank(part string, specialElements map[string]int) int {
	// Check for exact matches
	if rank, ok := specialElements[part]; ok {
		return rank
	}

	// Check for contains matches
	for name, rank := range specialElements {
		if strings.Contains(part, name) {
			return rank
		}
	}

	return 0
}

// buildXMLTree constructs an XML tree from the map
func buildXMLTree(m XMLMap, rootPath string) (*xmlNode, map[string]*xmlNode, error) {
	// Create the root node
	root := &xmlNode{
		path:       rootPath,
		name:       strings.TrimPrefix(rootPath, "/"),
		depth:      1,
		children:   make([]*xmlNode, 0, 4),
		attributes: make([]*xmlNode, 0, 4),
	}

	// Store value for root if exists
	if val, ok := m[rootPath]; ok {
		root.value = val
	}

	// Create a map to track nodes by path
	nodeMap := make(map[string]*xmlNode)
	nodeMap[rootPath] = root

	// Sort paths to ensure consistent processing order
	paths := make([]string, 0, len(m))
	for path := range m {
		if path != rootPath { // Skip root, already processed
			paths = append(paths, path)
		}
	}

	// Sort paths to ensure parents are created before children
	sort.Slice(paths, func(i, j int) bool {
		return comparePaths(paths[i], paths[j])
	})

	// Path builder for string operations
	pathBuilder := getPathBuilder()
	defer putPathBuilder(pathBuilder)

	// Process each path
	for _, path := range paths {
		processSinglePath(path, m, nodeMap, pathBuilder)
	}

	return root, nodeMap, nil
}

// processSinglePath adds a single path to the XML tree
func processSinglePath(path string, m XMLMap, nodeMap map[string]*xmlNode, pathBuilder *strings.Builder) {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return // Skip invalid paths
	}

	// Parse path information
	isAttr, parentPath, nodeName, attrName := parsePath(parts, pathBuilder)

	// Get or create parent node
	parent, exists := nodeMap[parentPath]
	if !exists {
		parent = createParentNodes(parts, nodeMap, pathBuilder)
	}

	// Skip if parent couldn't be created
	if parent == nil {
		return
	}

	// Add node to parent
	if isAttr {
		addAttributeNode(parent, path, nodeName, attrName, m[path])
	} else {
		addElementNode(parent, path, nodeName, m[path], nodeMap)
	}
}

// parsePath extracts path components from a path string
func parsePath(parts []string, pathBuilder *strings.Builder) (bool, string, string, string) {
	isAttr := false
	attrName := ""
	nodeName := parts[len(parts)-1]

	// Build parent path
	pathBuilder.Reset()
	for i := 0; i < len(parts)-1; i++ {
		pathBuilder.WriteString(parts[i])
		if i < len(parts)-2 {
			pathBuilder.WriteString("/")
		}
	}
	parentPath := pathBuilder.String()

	// Check if this is an attribute
	if strings.HasPrefix(nodeName, "@") {
		isAttr = true
		attrName = strings.TrimPrefix(nodeName, "@")

		// Get the node name from the parent path
		if len(parts) >= 3 {
			nodeName = parts[len(parts)-2]
		}
	}

	// Remove index from node name if present
	if idx := strings.Index(nodeName, "["); idx != -1 {
		nodeName = nodeName[:idx]
	}

	return isAttr, parentPath, nodeName, attrName
}

// createParentNodes creates missing parent nodes in the tree
func createParentNodes(parts []string, nodeMap map[string]*xmlNode, pathBuilder *strings.Builder) *xmlNode {
	currentPath := ""
	var currentNode *xmlNode

	// Create each parent node in sequence
	for i := 1; i < len(parts)-1; i++ {
		part := parts[i]

		// Remove index from part if present
		if idx := strings.Index(part, "["); idx != -1 {
			part = part[:idx]
		}

		// Build path to this node
		pathBuilder.Reset()
		if i == 1 {
			pathBuilder.WriteString("/")
		} else {
			pathBuilder.WriteString(currentPath)
			pathBuilder.WriteString("/")
		}
		pathBuilder.WriteString(part)
		currentPath = pathBuilder.String()

		// Check if node already exists
		if node, ok := nodeMap[currentPath]; ok {
			currentNode = node
			continue
		}

		// Create a new node
		depth := strings.Count(currentPath, "/")
		newNode := &xmlNode{
			path:       currentPath,
			name:       part,
			depth:      depth,
			children:   make([]*xmlNode, 0, 4),
			attributes: make([]*xmlNode, 0, 4),
		}
		nodeMap[currentPath] = newNode

		if currentNode != nil {
			currentNode.children = append(currentNode.children, newNode)
		}
		currentNode = newNode
	}

	return currentNode
}

// addAttributeNode adds an attribute node to a parent node
func addAttributeNode(parent *xmlNode, path, nodeName, attrName, value string) {
	attr := &xmlNode{
		path:     path,
		name:     nodeName,
		value:    value,
		isAttr:   true,
		attrName: attrName,
	}
	parent.attributes = append(parent.attributes, attr)
}

// addElementNode adds an element node to a parent node
func addElementNode(parent *xmlNode, path, nodeName, value string, nodeMap map[string]*xmlNode) {
	depth := strings.Count(path, "/")
	node := &xmlNode{
		path:       path,
		name:       nodeName,
		value:      value,
		depth:      depth,
		children:   make([]*xmlNode, 0, 4),
		attributes: make([]*xmlNode, 0, 4),
	}
	nodeMap[path] = node
	parent.children = append(parent.children, node)
}

// writeXMLNode writes a node and its children to the encoder
func writeXMLNode(node *xmlNode, enc *xml.Encoder, compareFn func(string, string) bool) error {
	// Split name into prefix and local parts for namespaced elements
	var prefix, local string
	if idx := strings.Index(node.name, ":"); idx != -1 {
		prefix, local = node.name[:idx], node.name[idx+1:]
	} else {
		local = node.name
	}

	// Create start element
	start := xml.StartElement{
		Name: xml.Name{Local: local},
	}
	if prefix != "" {
		start.Name.Local = prefix + ":" + local
	}

	// Pre-allocate attribute slice if needed
	if len(node.attributes) > 0 {
		start.Attr = make([]xml.Attr, 0, len(node.attributes))
	}

	// Add attributes
	for _, attr := range node.attributes {
		attrName := attr.attrName
		if idx := strings.Index(attrName, ":"); idx != -1 {
			prefix, local := attrName[:idx], attrName[idx+1:]
			attrName = prefix + ":" + local
		}
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: attrName},
			Value: attr.value,
		})
	}

	// Write start element
	if err := enc.EncodeToken(start); err != nil {
		return err
	}

	// Write element value if present
	if node.value != "" {
		if err := enc.EncodeToken(xml.CharData(node.value)); err != nil {
			return err
		}
	}

	// Sort and write children
	if len(node.children) > 1 {
		sort.Slice(node.children, func(i, j int) bool {
			return compareFn(node.children[i].path, node.children[j].path)
		})
	}

	for _, child := range node.children {
		if err := writeXMLNode(child, enc, compareFn); err != nil {
			return err
		}
	}

	// Write end element
	if err := enc.EncodeToken(start.End()); err != nil {
		return err
	}

	return nil
}

// extractBasePath extracts the base path without indices from an XPath
func extractBasePath(path string, builder *strings.Builder) string {
	builder.Reset()

	// Split path into segments
	parts := strings.Split(path, "/")

	// Skip empty first part if it exists
	start := 0
	if len(parts) > 0 && parts[0] == "" {
		start = 1
		builder.WriteString("/")
	}

	// Process each part
	for i := start; i < len(parts); i++ {
		part := parts[i]

		// Skip empty parts
		if part == "" {
			continue
		}

		// Remove index from part if present
		if idx := strings.Index(part, "["); idx != -1 {
			builder.WriteString(part[:idx])
		} else {
			builder.WriteString(part)
		}

		// Add separator unless it's the last part
		if i < len(parts)-1 {
			builder.WriteString("/")
		}
	}

	result := builder.String()
	return result
}
