package server

import (
	"testing"
	"strings"
)

func testHeader(t *testing.T, header string, expected []string) {
	split := splitHeader(header);
	if strings.Join(split, "#") != strings.Join(expected, "#") {
		t.Errorf("Failed to split %#v:\n  split:  %#v\n  expect: %#v\n", header, split, expected);
	}
}

func TestSplitHeaders(t *testing.T) {
	testHeader(t, `a, b`, []string{`a`, ` b`})
	testHeader(t, `a, b,`, []string{`a`, ` b`})
	testHeader(t, `a, b, `, []string{`a`, ` b`, ` `})
	testHeader(t, `a, b; test="a,b"; q=3.5, d`, []string{`a`, ` b; test="a,b"; q=3.5`, ` d`})
	testHeader(t, `a, b; test="a,b\"c"; q=3.5, d`, []string{`a`, ` b; test="a,b\"c"; q=3.5`, ` d`})
	testHeader(t, `a, b; test="a,b\"c,d\\e,f\"g,h"; q=3.5, d`, []string{`a`, ` b; test="a,b\"c,d\\e,f\"g,h"; q=3.5`, ` d`})
}