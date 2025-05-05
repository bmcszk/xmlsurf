// Package xmlsurf provides a modern and efficient way to work with XML in Go.
package xmlsurf

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
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

type elementContext struct {
	name       string
	fullPath   string
	namespaces map[string]string // prefix -> URI
}

// XMLMap represents a map of XPath expressions to their values
type XMLMap map[string][]string

// Equal returns true if both XMLMaps have the same XPaths and values in the same order
func (m XMLMap) Equal(other XMLMap) bool {
	if len(m) != len(other) {
		return false
	}
	for xpath, values := range m {
		otherValues, exists := other[xpath]
		if !exists {
			return false
		}
		if len(values) != len(otherValues) {
			return false
		}
		for i, value := range values {
			if value != otherValues[i] {
				return false
			}
		}
	}
	return true
}

// EqualIgnoreOrder returns true if both XMLMaps have the same XPaths and values, regardless of order
func (m XMLMap) EqualIgnoreOrder(other XMLMap) bool {
	if len(m) != len(other) {
		return false
	}
	for xpath, values := range m {
		otherValues, exists := other[xpath]
		if !exists {
			return false
		}
		if len(values) != len(otherValues) {
			return false
		}
		// Create maps to count occurrences of each value
		valueCount := make(map[string]int)
		otherValueCount := make(map[string]int)
		for _, value := range values {
			valueCount[value]++
		}
		for _, value := range otherValues {
			otherValueCount[value]++
		}
		// Compare value counts
		for value, count := range valueCount {
			if otherValueCount[value] != count {
				return false
			}
		}
	}
	return true
}

// ParseToMap parses XML from the given reader and returns an XMLMap of XPath expressions to their values.
// Options can be provided to configure parsing behavior. If no options are provided, default options will be used.
func ParseToMap(reader io.Reader, opts ...Option) (XMLMap, error) {
	if reader == nil {
		return nil, errors.New("reader cannot be nil")
	}

	// Apply default options
	options := DefaultParseOptions()
	// Apply any provided options
	for _, opt := range opts {
		opt(options)
	}

	result := make(XMLMap)
	decoder := xml.NewDecoder(reader)

	// Stack to keep track of current path context
	var pathStack []elementContext
	// Track if we've seen a root element
	hasRoot := false

	// Helper function to find namespace prefix for a URI
	findPrefix := func(uri string, ctx elementContext) string {
		for prefix, nsURI := range ctx.namespaces {
			if nsURI == uri {
				return prefix
			}
		}
		// Look in parent contexts
		for i := len(pathStack) - 1; i >= 0; i-- {
			for prefix, nsURI := range pathStack[i].namespaces {
				if nsURI == uri {
					return prefix
				}
			}
		}
		return ""
	}

	// Helper function to get element name with or without namespace
	getElementName := func(name xml.Name, ctx elementContext) string {
		if !options.IncludeNamespaces || name.Space == "" {
			return name.Local
		}
		if prefix := findPrefix(name.Space, ctx); prefix != "" {
			return prefix + ":" + name.Local
		}
		return name.Local
	}

	// Helper function to transform value if a transform function is set
	transformValue := func(value string) string {
		if options.ValueTransform != nil {
			return options.ValueTransform(value)
		}
		return value
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				if len(pathStack) > 0 {
					return nil, fmt.Errorf("XML syntax error: unclosed elements")
				}
				break
			}
			return nil, fmt.Errorf("XML syntax error: %v", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Check for multiple roots
			if len(pathStack) == 0 && hasRoot {
				return nil, fmt.Errorf("XML syntax error: multiple root elements")
			}
			if len(pathStack) == 0 {
				hasRoot = true
			}

			// Build current path
			parentPath := ""
			currentNamespaces := make(map[string]string)

			// Inherit parent namespaces
			if len(pathStack) > 0 {
				parentPath = pathStack[len(pathStack)-1].fullPath
				for prefix, uri := range pathStack[len(pathStack)-1].namespaces {
					currentNamespaces[prefix] = uri
				}
			}

			// Process namespace declarations
			for _, attr := range t.Attr {
				if attr.Name.Space == "xmlns" {
					currentNamespaces[attr.Name.Local] = attr.Value
				} else if attr.Name.Local == "xmlns" {
					currentNamespaces[""] = attr.Value
				}
			}

			// Create context for this element
			ctx := elementContext{
				name:       t.Name.Local,
				namespaces: currentNamespaces,
			}

			// Get element name with or without namespace
			elementName := getElementName(t.Name, ctx)
			currentPath := parentPath + "/" + elementName
			ctx.fullPath = currentPath

			// Push current element context to stack
			pathStack = append(pathStack, ctx)

			// Handle attributes
			for _, attr := range t.Attr {
				if attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns" {
					continue // Skip namespace declarations
				}
				attrName := attr.Name.Local
				if options.IncludeNamespaces && attr.Name.Space != "" {
					if prefix := findPrefix(attr.Name.Space, ctx); prefix != "" {
						attrName = prefix + ":" + attrName
					}
				}
				attrPath := currentPath + "/@" + attrName
				result[attrPath] = append(result[attrPath], transformValue(attr.Value))
			}

		case xml.EndElement:
			if len(pathStack) > 0 {
				pathStack = pathStack[:len(pathStack)-1]
			}

		case xml.CharData:
			if len(pathStack) > 0 {
				content := strings.TrimSpace(string(t))
				if content != "" {
					currentPath := pathStack[len(pathStack)-1].fullPath
					result[currentPath] = append(result[currentPath], transformValue(content))
				}
			}
		}
	}

	if !hasRoot {
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

	type xmlNode struct {
		path       string
		name       string
		value      string
		isAttr     bool
		attrName   string
		children   []*xmlNode
		attributes []*xmlNode
	}

	// Create a tree structure
	root := &xmlNode{path: rootPath, name: strings.TrimPrefix(rootPath, "/")}
	nodeMap := make(map[string]*xmlNode)
	nodeMap[rootPath] = root

	// Sort paths to ensure consistent processing order
	paths := make([]string, 0, len(m))
	for path := range m {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	// Build the tree
	for _, path := range paths {
		if path == rootPath {
			if len(m[path]) > 0 {
				root.value = m[path][0]
			}
			continue
		}

		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			continue
		}

		isAttr := false
		attrName := ""
		nodeName := parts[len(parts)-1]
		parentPath := strings.Join(parts[:len(parts)-1], "/")

		// Handle attributes
		if strings.HasPrefix(nodeName, "@") {
			isAttr = true
			attrName = strings.TrimPrefix(nodeName, "@")
			nodeName = parts[len(parts)-2]
		}

		parent, exists := nodeMap[parentPath]
		if !exists {
			// Create missing parent nodes
			currentPath := ""
			var currentNode *xmlNode
			for _, part := range parts[1 : len(parts)-1] {
				currentPath += "/" + part
				if node, ok := nodeMap[currentPath]; ok {
					currentNode = node
					continue
				}
				newNode := &xmlNode{
					path: currentPath,
					name: part,
				}
				nodeMap[currentPath] = newNode
				if currentNode != nil {
					currentNode.children = append(currentNode.children, newNode)
				}
				currentNode = newNode
			}
			parent = currentNode
		}

		if isAttr {
			for _, val := range m[path] {
				attr := &xmlNode{
					path:     path,
					name:     nodeName,
					value:    val,
					isAttr:   true,
					attrName: attrName,
				}
				parent.attributes = append(parent.attributes, attr)
			}
		} else {
			for _, val := range m[path] {
				node := &xmlNode{
					path:  path,
					name:  nodeName,
					value: val,
				}
				nodeMap[path] = node
				if parent != nil {
					parent.children = append(parent.children, node)
				}
			}
		}
	}

	// Write XML
	enc := xml.NewEncoder(w)
	if indent {
		enc.Indent("", "  ")
	}

	var writeNode func(*xmlNode) error
	writeNode = func(node *xmlNode) error {
		// Split name into prefix and local parts for namespaced elements
		var prefix, local string
		if parts := strings.Split(node.name, ":"); len(parts) > 1 {
			prefix, local = parts[0], parts[1]
		} else {
			local = node.name
		}

		start := xml.StartElement{
			Name: xml.Name{
				Space: prefix,
				Local: local,
			},
		}

		// Add attributes
		for _, attr := range node.attributes {
			var attrPrefix, attrLocal string
			if parts := strings.Split(attr.attrName, ":"); len(parts) > 1 {
				attrPrefix, attrLocal = parts[0], parts[1]
			} else {
				attrLocal = attr.attrName
			}

			start.Attr = append(start.Attr, xml.Attr{
				Name: xml.Name{
					Space: attrPrefix,
					Local: attrLocal,
				},
				Value: attr.value,
			})
		}

		if err := enc.EncodeToken(start); err != nil {
			return err
		}

		if node.value != "" {
			if err := enc.EncodeToken(xml.CharData(node.value)); err != nil {
				return err
			}
		}

		// Sort children for consistent output
		sort.Slice(node.children, func(i, j int) bool {
			return node.children[i].path < node.children[j].path
		})

		for _, child := range node.children {
			if err := writeNode(child); err != nil {
				return err
			}
		}

		return enc.EncodeToken(start.End())
	}

	if err := writeNode(root); err != nil {
		return err
	}

	return enc.Flush()
}
