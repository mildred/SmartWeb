package nquads

import (
	"fmt"
	"github.com/mildred/SmartWeb/turtle"
	"net/url"
)

var XsdNamespace = `http://www.w3.org/2001/XMLSchema#`

type BlankNode struct {
	string
}

type IriNode struct {
	string
}

type StringNode struct {
	string
}

type LocStringNode struct {
	StringNode
	Lang string
}

type TypedStringNode struct {
	StringNode
	Type IriNode
}

func EscapeString (s string) string {
	return turtle.Escape(s, turtle.EscapeStringMoreChars)
}

func EscapeIri (s string) string {
	return turtle.Escape(s, turtle.EscapeIriChars)
}

func EncodeString(s string) string {
	return `"` + EscapeString(s) + `"`
}

func EncodeLocString(s, lang string) string {
	return EncodeString(s) + "@" + lang
}

func EncodeTypedString(s, typeIri string) string {
	return EncodeString(s) + "^^" + EncodeIri(typeIri)
}

func EncodeIri(iri string) string {
	return `<` + EscapeIri(iri) + `>`
}

func EncodeBlank(name string) string {
	return "_:" + name
}

func EncodeBoolean(b bool) string {
	if b {
		return EncodeTypedString("true", XsdNamespace + "boolean")
	} else {
		return EncodeTypedString("false", XsdNamespace + "boolean")
	}
}

func EncodeInteger(i interface{}) string {
	return EncodeTypedString(fmt.Sprintf("%d", i), XsdNamespace + "number")
}

func EncodeAny(node interface{}) string {
	switch n := node.(type) {
		case BlankNode:       return EncodeBlank(n.string)
		case IriNode:         return EncodeIri(n.string)
		case LocStringNode:   return EncodeLocString(n.string, n.Lang)
		case TypedStringNode: return EncodeTypedString(n.string, n.Type.string)
		case StringNode:      return EncodeString(n.string)
		case string:          return EncodeString(n)
		case *url.URL:        return EncodeIri(n.String())
		case bool:            return EncodeBoolean(n)
		case int:	          return EncodeInteger(n)
		case int8:	          return EncodeInteger(n)
		case int16:           return EncodeInteger(n)
		case int32:           return EncodeInteger(n)
		case int64:           return EncodeInteger(n)
		case uint:            return EncodeInteger(n)
		case uint8:           return EncodeInteger(n)
		case uint16:          return EncodeInteger(n)
		case uint32:          return EncodeInteger(n)
		case uint64:          return EncodeInteger(n)
		default:              return ""
	}
}

func EncodeIriOrBlank(node interface{}) string {
	switch n := node.(type) {
		case BlankNode:       return EncodeBlank(n.string)
		case IriNode:         return EncodeIri(n.string)
		case string:          return EncodeIri(n)
		case *url.URL:        return EncodeIri(n.String())
		default:              return ""
	}
}

func EncodeIriInterface(node interface{}) string {
	switch n := node.(type) {
		case IriNode:         return EncodeIri(n.string)
		case string:          return EncodeIri(n)
		case *url.URL:        return EncodeIri(n.String())
		default:              return ""
	}
}