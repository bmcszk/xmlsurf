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
				"/root": "value",
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
				"/root/child":          "child value",
				"/root/another/nested": "nested value",
			},
		},
		{
			name: "elements with attributes",
			xml: `<root>
				<item id="1">first</item>
				<item id="2">second</item>
			</root>`,
			expected: XMLMap{
				"/root/item[1]":     "first",
				"/root/item[2]":     "second",
				"/root/item[1]/@id": "1",
				"/root/item[2]/@id": "2",
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
				"/root/items/item[1]": "one",
				"/root/items/item[2]": "two",
				"/root/items/item[3]": "three",
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
				"/root/items/item[1]/name":          "Product 1",
				"/root/items/item[2]/name":          "Product 2",
				"/root/items/item[1]/price":         "100",
				"/root/items/item[2]/price":         "200",
				"/root/items/item[1]/details/color": "red",
				"/root/items/item[2]/details/color": "blue",
				"/root/items/item[1]/details/size":  "large",
				"/root/items/item[2]/details/size":  "medium",
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
				"/soap:Envelope/soap:Header/ns1:AuthHeader/ns1:Username":                                            "john.doe",
				"/soap:Envelope/soap:Header/ns1:AuthHeader/ns1:Token":                                               "abc123",
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Category":                                             "Electronics",
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Products/ns2:Product/ns2:Name":                        "Laptop",
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Products/ns2:Product/ns2:Price":                       "999.99",
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Products/ns2:Product/ns2:Specs/ns3:Spec[1]/ns3:Name":  "CPU",
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Products/ns2:Product/ns2:Specs/ns3:Spec[2]/ns3:Name":  "RAM",
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Products/ns2:Product/ns2:Specs/ns3:Spec[1]/ns3:Value": "Intel i7",
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Products/ns2:Product/ns2:Specs/ns3:Spec[2]/ns3:Value": "16GB",
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
				"/Envelope/Header/AuthHeader/Username":                            "john.doe",
				"/Envelope/Header/AuthHeader/Token":                               "abc123",
				"/Envelope/Body/GetProducts/Category":                             "Electronics",
				"/Envelope/Body/GetProducts/Products/Product/Name":                "Laptop",
				"/Envelope/Body/GetProducts/Products/Product/Price":               "999.99",
				"/Envelope/Body/GetProducts/Products/Product/Specs/Spec[1]/Name":  "CPU",
				"/Envelope/Body/GetProducts/Products/Product/Specs/Spec[2]/Name":  "RAM",
				"/Envelope/Body/GetProducts/Products/Product/Specs/Spec[1]/Value": "Intel i7",
				"/Envelope/Body/GetProducts/Products/Product/Specs/Spec[2]/Value": "16GB",
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
				"/root/items/item[1]": "HELLO",
				"/root/items/item[2]": "WORLD",
				"/root/meta":          "INFO",
				"/root/meta/@id":      "TEST",
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
				"/root/items/item[1]": "hello!",
				"/root/items/item[2]": "world!",
				"/root/meta":          "info!",
				"/root/meta/@id":      "test!",
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
				"/root/items/item[1]": "HELLO",
				"/root/items/item[2]": "WORLD",
				"/root/meta":          "INFO",
				"/root/meta/@id":      "TEST",
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
			name: "equal maps",
			map1: XMLMap{
				"/root": "value",
			},
			map2: XMLMap{
				"/root": "value",
			},
			equal:        true,
			equalNoOrder: true,
		},
		{
			name: "different values",
			map1: XMLMap{
				"/root": "value1",
			},
			map2: XMLMap{
				"/root": "value2",
			},
			equal:        false,
			equalNoOrder: false,
		},
		{
			name: "different keys",
			map1: XMLMap{
				"/root1": "value",
			},
			map2: XMLMap{
				"/root2": "value",
			},
			equal:        false,
			equalNoOrder: false,
		},
		{
			name: "different sizes",
			map1: XMLMap{
				"/root1": "value1",
				"/root2": "value2",
			},
			map2: XMLMap{
				"/root1": "value1",
			},
			equal:        false,
			equalNoOrder: false,
		},
		{
			name: "nested arrays - equal",
			map1: XMLMap{
				"/root/items[1]/subItems[1]/name": "first",
				"/root/items[1]/subItems[2]/name": "second",
				"/root/items[2]/subItems[1]/name": "third",
				"/root/items[2]/subItems[2]/name": "fourth",
			},
			map2: XMLMap{
				"/root/items[1]/subItems[1]/name": "first",
				"/root/items[1]/subItems[2]/name": "second",
				"/root/items[2]/subItems[1]/name": "third",
				"/root/items[2]/subItems[2]/name": "fourth",
			},
			equal:        true,
			equalNoOrder: true,
		},
		{
			name: "nested arrays - different values",
			map1: XMLMap{
				"/root/items[1]/subItems[1]/name": "first",
				"/root/items[1]/subItems[2]/name": "second",
				"/root/items[2]/subItems[1]/name": "third",
				"/root/items[2]/subItems[2]/name": "fourth",
			},
			map2: XMLMap{
				"/root/items[1]/subItems[1]/name": "first",
				"/root/items[1]/subItems[2]/name": "changed",
				"/root/items[2]/subItems[1]/name": "third",
				"/root/items[2]/subItems[2]/name": "fourth",
			},
			equal:        false,
			equalNoOrder: false,
		},
		{
			name: "nested arrays - different structure",
			map1: XMLMap{
				"/root/items[1]/subItems[1]/name": "first",
				"/root/items[1]/subItems[2]/name": "second",
			},
			map2: XMLMap{
				"/root/items[1]/subItems[1]/name": "first",
				"/root/items[1]/subItems[2]/name": "second",
				"/root/items[2]/subItems[1]/name": "extra",
			},
			equal:        false,
			equalNoOrder: false,
		},
		{
			name: "nested arrays with attributes",
			map1: XMLMap{
				"/root/items[1]/@type":            "group1",
				"/root/items[1]/subItems[1]/@id":  "1",
				"/root/items[1]/subItems[1]/name": "first",
				"/root/items[1]/subItems[2]/@id":  "2",
				"/root/items[1]/subItems[2]/name": "second",
				"/root/items[2]/@type":            "group2",
				"/root/items[2]/subItems[1]/@id":  "3",
				"/root/items[2]/subItems[1]/name": "third",
				"/root/items[2]/subItems[2]/@id":  "4",
				"/root/items[2]/subItems[2]/name": "fourth",
			},
			map2: XMLMap{
				"/root/items[1]/@type":            "group1",
				"/root/items[1]/subItems[1]/@id":  "1",
				"/root/items[1]/subItems[1]/name": "first",
				"/root/items[1]/subItems[2]/@id":  "2",
				"/root/items[1]/subItems[2]/name": "second",
				"/root/items[2]/@type":            "group2",
				"/root/items[2]/subItems[1]/@id":  "3",
				"/root/items[2]/subItems[1]/name": "third",
				"/root/items[2]/subItems[2]/@id":  "4",
				"/root/items[2]/subItems[2]/name": "fourth",
			},
			equal:        true,
			equalNoOrder: true,
		},
		{
			name: "deeply nested arrays",
			map1: XMLMap{
				"/root/level1[1]/level2[1]/level3[1]/value": "a",
				"/root/level1[1]/level2[1]/level3[2]/value": "b",
				"/root/level1[1]/level2[2]/level3[1]/value": "c",
				"/root/level1[2]/level2[1]/level3[1]/value": "d",
			},
			map2: XMLMap{
				"/root/level1[1]/level2[1]/level3[1]/value": "a",
				"/root/level1[1]/level2[1]/level3[2]/value": "b",
				"/root/level1[1]/level2[2]/level3[1]/value": "c",
				"/root/level1[2]/level2[1]/level3[1]/value": "d",
			},
			equal:        true,
			equalNoOrder: true,
		},
		{
			name: "mixed array depths",
			map1: XMLMap{
				"/root/simple":                           "value",
				"/root/array[1]":                         "first",
				"/root/array[2]":                         "second",
				"/root/nested[1]/items[1]/deep[1]/value": "a",
				"/root/nested[1]/items[1]/deep[2]/value": "b",
				"/root/nested[2]/items[1]/deep[1]/value": "c",
			},
			map2: XMLMap{
				"/root/simple":                           "value",
				"/root/array[1]":                         "first",
				"/root/array[2]":                         "second",
				"/root/nested[1]/items[1]/deep[1]/value": "a",
				"/root/nested[1]/items[1]/deep[2]/value": "b",
				"/root/nested[2]/items[1]/deep[1]/value": "c",
			},
			equal:        true,
			equalNoOrder: true,
		},
		{
			name: "same values different order - simple",
			map1: XMLMap{
				"/root/items[1]": "a",
				"/root/items[2]": "b",
				"/root/items[3]": "c",
			},
			map2: XMLMap{
				"/root/items[1]": "c",
				"/root/items[2]": "a",
				"/root/items[3]": "b",
			},
			equal:        false,
			equalNoOrder: true,
		},
		{
			name: "same values different order - nested",
			map1: XMLMap{
				"/root/items[1]/value": "a",
				"/root/items[2]/value": "b",
				"/root/items[3]/value": "c",
			},
			map2: XMLMap{
				"/root/items[1]/value": "c",
				"/root/items[2]/value": "a",
				"/root/items[3]/value": "b",
			},
			equal:        false,
			equalNoOrder: true,
		},
		{
			name: "same values different order - deep nested",
			map1: XMLMap{
				"/root/level1[1]/level2[1]/value": "a",
				"/root/level1[1]/level2[2]/value": "b",
				"/root/level1[2]/level2[1]/value": "c",
			},
			map2: XMLMap{
				"/root/level1[1]/level2[1]/value": "c",
				"/root/level1[1]/level2[2]/value": "a",
				"/root/level1[2]/level2[1]/value": "b",
			},
			equal:        false,
			equalNoOrder: true,
		},
		{
			name: "nested arrays same values different order",
			map1: XMLMap{
				"/root/items[1]/subItems[1]/name": "first",
				"/root/items[1]/subItems[2]/name": "second",
				"/root/items[2]/subItems[1]/name": "third",
				"/root/items[2]/subItems[2]/name": "fourth",
			},
			map2: XMLMap{
				"/root/items[1]/subItems[1]/name": "third",
				"/root/items[1]/subItems[2]/name": "fourth",
				"/root/items[2]/subItems[1]/name": "first",
				"/root/items[2]/subItems[2]/name": "second",
			},
			equal:        false,
			equalNoOrder: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.map1.Equal(tt.map2); got != tt.equal {
				t.Errorf("Equal() = %v, want %v", got, tt.equal)
			}
			if got := tt.map1.EqualIgnoreOrder(tt.map2); got != tt.equalNoOrder {
				t.Errorf("EqualIgnoreOrder() = %v, want %v", got, tt.equalNoOrder)
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
			name:        "empty input",
			xml:         "",
			expectedErr: "EOF",
		},
		{
			name:        "invalid xml",
			xml:         "<root>",
			expectedErr: "XML syntax error on line 1: unexpected EOF",
		},
		{
			name:        "multiple root elements",
			xml:         "<root1></root1><root2></root2>",
			expectedErr: "XML syntax error: multiple root elements",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.xml)
			_, err := ParseToMap(reader)
			if err == nil {
				t.Errorf("ParseToMap() expected error %q, got nil", tt.expectedErr)
				return
			}
			if err.Error() != tt.expectedErr {
				t.Errorf("ParseToMap() error = %q, want %q", err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestXMLMapToXML(t *testing.T) {
	tests := []struct {
		name     string
		input    XMLMap
		expected string
	}{
		{
			name: "simple xml",
			input: XMLMap{
				"/root": "value",
			},
			expected: "<root>value</root>",
		},
		{
			name: "nested elements",
			input: XMLMap{
				"/root/child":          "child value",
				"/root/another/nested": "nested value",
			},
			expected: "<root><child>child value</child><another><nested>nested value</nested></another></root>",
		},
		{
			name: "elements with attributes",
			input: XMLMap{
				"/root/item[1]":     "first",
				"/root/item[2]":     "second",
				"/root/item[1]/@id": "1",
				"/root/item[2]/@id": "2",
			},
			expected: "<root><item id=\"1\">first</item><item id=\"2\">second</item></root>",
		},
		{
			name: "multiple elements with same name",
			input: XMLMap{
				"/root/items/item[1]": "one",
				"/root/items/item[2]": "two",
				"/root/items/item[3]": "three",
			},
			expected: "<root><items><item>one</item><item>two</item><item>three</item></items></root>",
		},
		{
			name: "xml with namespaces",
			input: XMLMap{
				"/soap:Envelope/soap:Header/ns1:AuthHeader/ns1:Username":                      "john.doe",
				"/soap:Envelope/soap:Header/ns1:AuthHeader/ns1:Token":                         "abc123",
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Category":                       "Electronics",
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Products/ns2:Product/ns2:Name":  "Laptop",
				"/soap:Envelope/soap:Body/ns2:GetProducts/ns2:Products/ns2:Product/ns2:Price": "999.99",
			},
			expected: "<soap:Envelope><soap:Header><ns1:AuthHeader><ns1:Username>john.doe</ns1:Username><ns1:Token>abc123</ns1:Token></ns1:AuthHeader></soap:Header><soap:Body><ns2:GetProducts><ns2:Category>Electronics</ns2:Category><ns2:Products><ns2:Product><ns2:Name>Laptop</ns2:Name><ns2:Price>999.99</ns2:Price></ns2:Product></ns2:Products></ns2:GetProducts></soap:Body></soap:Envelope>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			err := tt.input.ToXML(&builder, false)
			if err != nil {
				t.Errorf("ToXML() error = %v", err)
				return
			}

			result := builder.String()
			if result != tt.expected {
				t.Errorf("ToXML() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestXMLMapToXMLErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       XMLMap
		expectedErr string
	}{
		{
			name:        "empty map",
			input:       XMLMap{},
			expectedErr: "empty XMLMap",
		},
		{
			name: "invalid path",
			input: XMLMap{
				"invalid": "value",
			},
			expectedErr: "no root element found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			err := tt.input.ToXML(&builder, false)
			if err == nil {
				t.Errorf("ToXML() expected error %q, got nil", tt.expectedErr)
				return
			}
			if err.Error() != tt.expectedErr {
				t.Errorf("ToXML() error = %q, want %q", err.Error(), tt.expectedErr)
			}
		})
	}
}
