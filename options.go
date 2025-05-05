package xmlsurf

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
