package htmlutil

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestTemplates(t *testing.T) {
	tests := []struct {
		name     string
		data     Element
		expected string
	}{
		{
			"Self-closing without attributes",
			Element{Tag: "input"},
			"<input />",
		},
		{
			"Self-closing with attributes",
			Element{Tag: "input", Attributes: map[string]string{"type": "text", "value": "example"}},
			`<input type="text" value="example" />`,
		},
		{
			"Self-closing with boolean attributes",
			Element{Tag: "input", BooleanAttributes: []string{"checked"}},
			`<input checked />`,
		},
		{
			"Self-closing with both attributes",
			Element{Tag: "input", Attributes: map[string]string{"type": "text"}, BooleanAttributes: []string{"checked"}},
			`<input type="text" checked />`,
		},
		{
			"Non-self-closing without attributes",
			Element{Tag: "div", InnerHTML: "Hello"},
			`<div>Hello</div>`,
		},
		{
			"Non-self-closing with attributes",
			Element{Tag: "div", Attributes: map[string]string{"id": "main", "class": "container"}, InnerHTML: "Hello"},
			`<div id="main" class="container">Hello</div>`,
		},
		{
			"Non-self-closing with boolean attributes",
			Element{Tag: "div", BooleanAttributes: []string{"hidden"}, InnerHTML: "Hello"},
			`<div hidden>Hello</div>`,
		},
		{
			"Non-self-closing with both attributes",
			Element{Tag: "div", Attributes: map[string]string{"id": "main"}, BooleanAttributes: []string{"hidden"}, InnerHTML: "Hello"},
			`<div id="main" hidden>Hello</div>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Render the template to get the actual result.
			result, err := RenderElement(&tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Parse both the expected and actual HTML.
			expectedNode, err := parseHTML(tt.expected)
			if err != nil {
				t.Fatalf("error parsing expected HTML: %v", err)
			}
			resultNode, err := parseHTML(string(result))
			if err != nil {
				t.Fatalf("error parsing result HTML: %v", err)
			}

			// Check for double spaces in the output.
			if hasDoubleSpaces(string(result)) {
				t.Errorf("output contains double spaces: %s", result)
			}

			// Compare the parsed nodes structurally (ignoring attribute order).
			if !compareNodes(expectedNode, resultNode) {
				t.Errorf("expected HTML structure does not match actual structure.\nExpected: %s\nGot: %s", tt.expected, result)
			}
		})
	}
}

// Helper function to parse an HTML string into a node.
func parseHTML(input string) (*html.Node, error) {
	return html.Parse(strings.NewReader(input))
}

// Helper function to check if two nodes are structurally equivalent (ignoring attribute order).
func compareNodes(n1, n2 *html.Node) bool {
	// Compare node types and tag names.
	if n1.Type != n2.Type || n1.Data != n2.Data {
		return false
	}

	// Compare attributes, ignoring order.
	if len(n1.Attr) != len(n2.Attr) {
		return false
	}
	attrMap1 := make(map[string]string)
	for _, a := range n1.Attr {
		attrMap1[a.Key] = a.Val
	}
	for _, a := range n2.Attr {
		if attrMap1[a.Key] != a.Val {
			return false
		}
	}

	// Compare children recursively.
	n1Child, n2Child := n1.FirstChild, n2.FirstChild
	for n1Child != nil && n2Child != nil {
		if !compareNodes(n1Child, n2Child) {
			return false
		}
		n1Child = n1Child.NextSibling
		n2Child = n2Child.NextSibling
	}
	return n1Child == nil && n2Child == nil
}

// Helper function to check for double spaces.
func hasDoubleSpaces(s string) bool {
	return strings.Contains(s, "  ")
}
