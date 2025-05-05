package xmlsurf

import (
	"encoding/xml"
	"sort"
	"strings"
)

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
