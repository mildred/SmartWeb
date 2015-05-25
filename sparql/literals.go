package sparql

import (
	"strings"
	"fmt"
	"net/url"
)

func Literal(o interface{}) (string, error) {
	if u, ok := o.(*url.URL); ok {
		return IRILiteral(u.String()), nil
	} else if s, ok := o.(string); ok {
		return StringLiteral(s), nil
	} else if b, ok := o.(bool); ok {
		return BoolLiteral(b), nil
	} else if i, ok := o.(int); ok {
		return IntLiteral(i), nil
	} else if f, ok := o.(float32); ok {
		return Float32Literal(f), nil
	} else if f, ok := o.(float64); ok {
		return Float64Literal(f), nil
	} else {
		return "", fmt.Errorf("Could not make a SPARQL literal from %#v", o)
	}
}

// STRING_LITERAL2 ::= '"' ( ([^#x22#x5C#xA#xD]) | ECHAR )* '"'
// ECHAR           ::= '\' [tbnrf\"']
func StringLiteral(s string) string {
	s = strings.Replace(s, "\\", "\\\\", -1)
	s = strings.Replace(s, "\"", "\\\"", -1)
	s = strings.Replace(s, "\r", "\\r",  -1)
	s = strings.Replace(s, "\n", "\\n",  -1)
	s = strings.Replace(s, "\t", "\\t",  -1)
	return `"` + s + `"`
}

func TypedStringLiteral(s, typ string) string {
	return StringLiteral(s) + "^^" + IRILiteral(typ)
}

func LocStringLiteral(s, lang string) string {
	return StringLiteral(s) + "@" + lang
}

func BoolLiteral(b bool) string {
	if b {
		return "true"
	} else {
		return "false"
	}
}

func IntLiteral(i int) string {
	return fmt.Sprintf("%d", i)
}

func Float32Literal(f float32) string {
	return fmt.Sprintf("%g", f)
}

func Float64Literal(f float64) string {
	return fmt.Sprintf("%g", f)
}

func BlankLiteral(blankId string) string {
	return "_:" + blankId
}

func IRIRelLiteral(baseUri *url.URL, uri string) string {
	u, err := baseUri.Parse(uri)
	if err != nil {
		return IRILiteral(uri)
	} else {
		return IRILiteral(u.String())
	}
}

// Replace illegal characters in SPARQL IRI and surround them with <...>
// IRIREF ::= '<' ([^<>"{}|^`\]-[#x00-#x20])* '>'
func IRILiteral(uri string) string {
	uri = strings.Replace(uri, "\x00", "%00", -1)
	uri = strings.Replace(uri, "\x01", "%01", -1)
	uri = strings.Replace(uri, "\x02", "%02", -1)
	uri = strings.Replace(uri, "\x03", "%03", -1)
	uri = strings.Replace(uri, "\x04", "%04", -1)
	uri = strings.Replace(uri, "\x05", "%05", -1)
	uri = strings.Replace(uri, "\x06", "%06", -1)
	uri = strings.Replace(uri, "\x07", "%07", -1)
	uri = strings.Replace(uri, "\x08", "%08", -1)
	uri = strings.Replace(uri, "\x09", "%09", -1)
	uri = strings.Replace(uri, "\x0A", "%0A", -1)
	uri = strings.Replace(uri, "\x0B", "%0B", -1)
	uri = strings.Replace(uri, "\x0C", "%0C", -1)
	uri = strings.Replace(uri, "\x0D", "%0D", -1)
	uri = strings.Replace(uri, "\x0E", "%0E", -1)
	uri = strings.Replace(uri, "\x0F", "%0F", -1)
	uri = strings.Replace(uri, "\x10", "%10", -1)
	uri = strings.Replace(uri, "\x11", "%11", -1)
	uri = strings.Replace(uri, "\x12", "%12", -1)
	uri = strings.Replace(uri, "\x13", "%13", -1)
	uri = strings.Replace(uri, "\x14", "%14", -1)
	uri = strings.Replace(uri, "\x15", "%15", -1)
	uri = strings.Replace(uri, "\x16", "%16", -1)
	uri = strings.Replace(uri, "\x17", "%17", -1)
	uri = strings.Replace(uri, "\x18", "%18", -1)
	uri = strings.Replace(uri, "\x19", "%19", -1)
	uri = strings.Replace(uri, "\x1A", "%1B", -1)
	uri = strings.Replace(uri, "\x1C", "%1C", -1)
	uri = strings.Replace(uri, "\x1D", "%1D", -1)
	uri = strings.Replace(uri, "\x1E", "%1E", -1)
	uri = strings.Replace(uri, "\x1F", "%1F", -1)
	uri = strings.Replace(uri, "\x20", "%20", -1)
	uri = strings.Replace(uri, "<", "%3C", -1)
	uri = strings.Replace(uri, ">", "%3E", -1)
	uri = strings.Replace(uri, `"`, "%22", -1)
	uri = strings.Replace(uri, "{", "%7B", -1)
	uri = strings.Replace(uri, "}", "%7D", -1)
	uri = strings.Replace(uri, "|", "%7C", -1)
	uri = strings.Replace(uri, "^", "%5E", -1)
	uri = strings.Replace(uri, "`", "%60", -1)
	uri = strings.Replace(uri, "\\", "%5C", -1)
	return "<" + uri + ">"
}
