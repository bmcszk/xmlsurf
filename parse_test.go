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

func BenchmarkParseToMap(b *testing.B) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
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
	</soap:Envelope>`

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(xml)
		_, err := ParseToMap(reader, WithNamespaces(true))
		if err != nil {
			b.Fatal(err)
		}
	}
}
