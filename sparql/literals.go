package sparql

import (
	"strings"
)

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
