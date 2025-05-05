package xmlsurf

import (
	"strings"
	"testing"
)

func TestParseXMLToMap(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		options  []Option
		expected XMLMap
	}{
		{
			name: "simple xml with single element",
			xml:  `<root>value</root>`,
			expected: XMLMap{
				"/root": {"value"},
			},
		},
		{
			name: "nested elements",
			xml: `<root>
				<child>child value</child>
				<another>
					<nested>nested value</nested>
				</another>
			</root>`,
			expected: XMLMap{
				"/root/child":          {"child value"},
				"/root/another/nested": {"nested value"},
			},
		},
		{
			name: "elements with attributes",
			xml: `<root>
				<item id="1">first</item>
				<item id="2">second</item>
			</root>`,
			expected: XMLMap{
				"/root/item":     {"first", "second"},
				"/root/item/@id": {"1", "2"},
			},
		},
		{
			name: "multiple elements with same name",
			xml: `<root>
				<items>
					<item>one</item>
					<item>two</item>
					<item>three</item>
				</items>
			</root>`,
			expected: XMLMap{
				"/root/items/item": {"one", "two", "three"},
			},
		},
		{
			name: "list items with nested elements",
			xml: `<root>
				<items>
					<item>
						<name>Product 1</name>
						<price>100</price>
						<details>
							<color>red</color>
							<size>large</size>
						</details>
					</item>
					<item>
						<name>Product 2</name>
						<price>200</price>
						<details>
							<color>blue</color>
							<size>medium</size>
						</details>
					</item>
				</items>
			</root>`,
			expected: XMLMap{
				"/root/items/item/name":          {"Product 1", "Product 2"},
				"/root/items/item/price":         {"100", "200"},
				"/root/items/item/details/color": {"red", "blue"},
				"/root/items/item/details/size":  {"large", "medium"},
			},
		},
		{
			name: "xml with namespaces - with namespaces",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
						  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
						  xmlns:xsd="http://www.w3.org/2001/XMLSchema">
				<soap:Header>
					<ns1:AuthHeader xmlns:ns1="http://example.com/auth">
						<ns1:Username>john.doe</ns1:Username>
						<ns1:Token>abc123</ns1:Token>
					</ns1:AuthHeader>
				</soap:Header>
				<soap:Body>
					<ns2:GetProducts xmlns:ns2="http://example.com/products">
						<ns2:Category>Electronics</ns2:Category>
						<ns2:Products>
							<ns2:Product>
								<ns2:Name>Laptop</ns2:Name>
								<ns2:Price>999.99</ns2:Price>
								<ns2:Specs>
									<ns3:Spec xmlns:ns3="http://example.com/specs">
										<ns3:Name>CPU</ns3:Name>
										<ns3:Value>Intel i7</ns3:Value>
									</ns3:Spec>
									<ns3:Spec xmlns:ns3="http://example.com/specs">
										<ns3:Name>RAM</ns3:Name>
										<ns3:Value>16GB</ns3:Value>
									</ns3:Spec>
								</ns2:Specs>
							</ns2:Product>
						</ns2:Products>
					</ns2:GetProducts>
				</soap:Body>
			</soap:Envelope>`,
			options: []Option{WithNamespaces(true)},
			expected: XMLMap{
				"/soap:Envelope/soap:Header/ns1:AuthHeader/ns1:Username":                                         {"john.doe"},
				"/soap:Envelope/soap:Header/ns1:AuthHeader/ns1:Token":                                            {"abc123"},
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Category":                                          {"Electronics"},
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Products/ns2:Product/ns2:Name":                     {"Laptop"},
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Products/ns2:Product/ns2:Price":                    {"999.99"},
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Products/ns2:Product/ns2:Specs/ns3:Spec/ns3:Name":  {"CPU", "RAM"},
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Products/ns2:Product/ns2:Specs/ns3:Spec/ns3:Value": {"Intel i7", "16GB"},
			},
		},
		{
			name: "xml with namespaces - without namespaces",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
						  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
						  xmlns:xsd="http://www.w3.org/2001/XMLSchema">
				<soap:Header>
					<ns1:AuthHeader xmlns:ns1="http://example.com/auth">
						<ns1:Username>john.doe</ns1:Username>
						<ns1:Token>abc123</ns1:Token>
					</ns1:AuthHeader>
				</soap:Header>
				<soap:Body>
					<ns2:GetProducts xmlns:ns2="http://example.com/products">
						<ns2:Category>Electronics</ns2:Category>
						<ns2:Products>
							<ns2:Product>
								<ns2:Name>Laptop</ns2:Name>
								<ns2:Price>999.99</ns2:Price>
								<ns2:Specs>
									<ns3:Spec xmlns:ns3="http://example.com/specs">
										<ns3:Name>CPU</ns3:Name>
										<ns3:Value>Intel i7</ns3:Value>
									</ns3:Spec>
									<ns3:Spec xmlns:ns3="http://example.com/specs">
										<ns3:Name>RAM</ns3:Name>
										<ns3:Value>16GB</ns3:Value>
									</ns3:Spec>
								</ns2:Specs>
							</ns2:Product>
						</ns2:Products>
					</ns2:GetProducts>
				</soap:Body>
			</soap:Envelope>`,
			options: []Option{WithNamespaces(false)},
			expected: XMLMap{
				"/Envelope/Header/AuthHeader/Username":                         {"john.doe"},
				"/Envelope/Header/AuthHeader/Token":                            {"abc123"},
				"/Envelope/Body/GetProducts/Category":                          {"Electronics"},
				"/Envelope/Body/GetProducts/Products/Product/Name":             {"Laptop"},
				"/Envelope/Body/GetProducts/Products/Product/Price":            {"999.99"},
				"/Envelope/Body/GetProducts/Products/Product/Specs/Spec/Name":  {"CPU", "RAM"},
				"/Envelope/Body/GetProducts/Products/Product/Specs/Spec/Value": {"Intel i7", "16GB"},
			},
		},
		{
			name: "xml with value transformation - uppercase",
			xml: `<root>
				<items>
					<item>hello</item>
					<item>world</item>
				</items>
				<meta id="test">info</meta>
			</root>`,
			options: []Option{
				WithValueTransform(strings.ToUpper),
			},
			expected: XMLMap{
				"/root/items/item": {"HELLO", "WORLD"},
				"/root/meta":       {"INFO"},
				"/root/meta/@id":   {"TEST"},
			},
		},
		{
			name: "xml with value transformation - custom",
			xml: `<root>
				<items>
					<item>  hello  </item>
					<item>  world  </item>
				</items>
				<meta id="  test  ">info</meta>
			</root>`,
			options: []Option{
				WithValueTransform(func(s string) string {
					return strings.TrimSpace(s) + "!"
				}),
			},
			expected: XMLMap{
				"/root/items/item": {"hello!", "world!"},
				"/root/meta":       {"info!"},
				"/root/meta/@id":   {"test!"},
			},
		},
		{
			name: "xml with multiple transformations",
			xml: `<root>
				<items>
					<item>  hello  </item>
					<item>  world  </item>
				</items>
				<meta id="  test  ">info</meta>
			</root>`,
			options: []Option{
				WithValueTransform(strings.TrimSpace),
				WithValueTransform(strings.ToUpper),
			},
			expected: XMLMap{
				"/root/items/item": {"HELLO", "WORLD"},
				"/root/meta":       {"INFO"},
				"/root/meta/@id":   {"TEST"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.xml)
			result, err := ParseToMap(reader, tt.options...)
			if err != nil {
				t.Errorf("ParseToMap() error = %v", err)
				return
			}

			if !result.Equal(tt.expected) {
				t.Errorf("ParseToMap() result = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestXMLMapComparison(t *testing.T) {
	tests := []struct {
		name         string
		map1         XMLMap
		map2         XMLMap
		equal        bool
		equalNoOrder bool
	}{
		{
			name: "identical maps",
			map1: XMLMap{
				"/root/item": {"one", "two", "three"},
			},
			map2: XMLMap{
				"/root/item": {"one", "two", "three"},
			},
			equal:        true,
			equalNoOrder: true,
		},
		{
			name: "different order",
			map1: XMLMap{
				"/root/item": {"one", "two", "three"},
			},
			map2: XMLMap{
				"/root/item": {"three", "one", "two"},
			},
			equal:        false,
			equalNoOrder: true,
		},
		{
			name: "different values",
			map1: XMLMap{
				"/root/item": {"one", "two", "three"},
			},
			map2: XMLMap{
				"/root/item": {"one", "two", "four"},
			},
			equal:        false,
			equalNoOrder: false,
		},
		{
			name: "different xpaths",
			map1: XMLMap{
				"/root/item1": {"one", "two"},
			},
			map2: XMLMap{
				"/root/item2": {"one", "two"},
			},
			equal:        false,
			equalNoOrder: false,
		},
		{
			name: "multiple xpaths",
			map1: XMLMap{
				"/root/item1": {"one", "two"},
				"/root/item2": {"three", "four"},
			},
			map2: XMLMap{
				"/root/item1": {"two", "one"},
				"/root/item2": {"four", "three"},
			},
			equal:        false,
			equalNoOrder: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.map1.Equal(tt.map2); got != tt.equal {
				t.Errorf("XMLMap.Equal() = %v, want %v", got, tt.equal)
			}
			if got := tt.map1.EqualIgnoreOrder(tt.map2); got != tt.equalNoOrder {
				t.Errorf("XMLMap.EqualIgnoreOrder() = %v, want %v", got, tt.equalNoOrder)
			}
		})
	}
}

func TestParseXMLToMapErrors(t *testing.T) {
	tests := []struct {
		name        string
		xml         string
		expectedErr string
	}{
		{
			name:        "invalid xml - unclosed tag",
			xml:         `<root><unclosed>`,
			expectedErr: "XML syntax error",
		},
		{
			name:        "empty input",
			xml:         "",
			expectedErr: "EOF",
		},
		{
			name:        "malformed attributes - unclosed quote",
			xml:         `<root attr="value>`,
			expectedErr: "XML syntax error",
		},
		{
			name:        "invalid xml - mismatched tags",
			xml:         `<root><child></root></child>`,
			expectedErr: "XML syntax error",
		},
		{
			name:        "invalid xml - invalid characters",
			xml:         `<root><child>&invalid;</child></root>`,
			expectedErr: "XML syntax error",
		},
		{
			name:        "invalid xml - multiple roots",
			xml:         `<root>value</root><another>value</another>`,
			expectedErr: "XML syntax error",
		},
		{
			name:        "invalid xml - invalid attribute syntax",
			xml:         `<root attr=value>content</root>`,
			expectedErr: "XML syntax error",
		},
		{
			name:        "invalid xml - invalid element name",
			xml:         `<root><1invalid>value</1invalid></root>`,
			expectedErr: "XML syntax error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.xml)
			_, err := ParseToMap(reader)
			if err == nil {
				t.Errorf("ParseToMap() expected error for input: %s", tt.xml)
				return
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("ParseToMap() error = %v, want error containing %q", err, tt.expectedErr)
			}
		})
	}
}

func TestXMLMapToXML(t *testing.T) {
	tests := []struct {
		name     string
		xmlMap   XMLMap
		indent   bool
		expected string
	}{
		{
			name: "simple element",
			xmlMap: XMLMap{
				"/root": {"value"},
			},
			indent:   false,
			expected: `<root>value</root>`,
		},
		{
			name: "nested elements",
			xmlMap: XMLMap{
				"/root/child":          {"child value"},
				"/root/another/nested": {"nested value"},
			},
			indent: true,
			expected: `<root>
  <child>child value</child>
  <another>
    <nested>nested value</nested>
  </another>
</root>`,
		},
		{
			name: "elements with attributes",
			xmlMap: XMLMap{
				"/root/item":     {"first", "second"},
				"/root/item/@id": {"1", "2"},
			},
			indent: true,
			expected: `<root>
  <item id="1">first</item>
  <item id="2">second</item>
</root>`,
		},
		{
			name: "namespaced elements",
			xmlMap: XMLMap{
				"/soap:Envelope/soap:Header/ns1:AuthHeader/ns1:Username": {"john.doe"},
				"/soap:Envelope/soap:Header/ns1:AuthHeader/ns1:Token":    {"abc123"},
			},
			indent: true,
			expected: `<soap:Envelope>
  <soap:Header>
    <ns1:AuthHeader>
      <ns1:Username>john.doe</ns1:Username>
      <ns1:Token>abc123</ns1:Token>
    </ns1:AuthHeader>
  </soap:Header>
</soap:Envelope>`,
		},
		{
			name: "complex structure",
			xmlMap: XMLMap{
				"/root/items/item/name":          {"Product 1", "Product 2"},
				"/root/items/item/price":         {"100", "200"},
				"/root/items/item/details/color": {"red", "blue"},
				"/root/items/item/details/size":  {"large", "medium"},
			},
			indent: true,
			expected: `<root>
  <items>
    <item>
      <name>Product 1</name>
      <price>100</price>
      <details>
        <color>red</color>
        <size>large</size>
      </details>
    </item>
    <item>
      <name>Product 2</name>
      <price>200</price>
      <details>
        <color>blue</color>
        <size>medium</size>
      </details>
    </item>
  </items>
</root>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			err := tt.xmlMap.ToXML(&buf, tt.indent)
			if err != nil {
				t.Errorf("ToXML() error = %v", err)
				return
			}

			got := buf.String()
			if got != tt.expected {
				t.Errorf("ToXML() got = %v, want %v", got, tt.expected)
			}

			// Verify that the generated XML can be parsed back
			reader := strings.NewReader(got)
			parsed, err := ParseToMap(reader)
			if err != nil {
				t.Errorf("ParseToMap() error = %v", err)
				return
			}

			if !parsed.Equal(tt.xmlMap) {
				t.Errorf("Round-trip conversion failed. Got = %v, want %v", parsed, tt.xmlMap)
			}
		})
	}
}

func TestXMLMapToXMLErrors(t *testing.T) {
	tests := []struct {
		name        string
		xmlMap      XMLMap
		expectedErr string
	}{
		{
			name:        "empty map",
			xmlMap:      XMLMap{},
			expectedErr: "empty XMLMap",
		},
		{
			name: "no root element",
			xmlMap: XMLMap{
				"invalid": {"value"},
			},
			expectedErr: "no root element found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			err := tt.xmlMap.ToXML(&buf, false)
			if err == nil {
				t.Error("ToXML() expected error")
				return
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("ToXML() error = %v, want error containing %q", err, tt.expectedErr)
			}
		})
	}
}
