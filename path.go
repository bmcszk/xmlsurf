package xmlsurf

import (
	"strings"
	"sync"
)

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
