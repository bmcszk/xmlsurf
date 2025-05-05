package xmlsurf

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ParseToMap parses XML from the reader and returns a map of XPath expressions to values.
// It accepts optional configuration through Option functions.
// The resulting map contains XPath expressions as keys and their corresponding values.
// For elements with attributes, the attribute paths are prefixed with "@".
// For repeated elements, indices are added to the path (e.g., /root/item[1], /root/item[2]).
func ParseToMap(reader io.Reader, opts ...Option) (XMLMap, error) {
	options := DefaultParseOptions()
	for _, opt := range opts {
		opt(options)
	}

	decoder := xml.NewDecoder(reader)
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
		pathBuilder.WriteString(":")
		pathBuilder.WriteString(elementName)
	} else {
		// For default namespace (no prefix), just return the element name
		pathBuilder.WriteString(elementName)
	}
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
