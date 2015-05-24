package nquads

import (
	"fmt"
)

type Statement struct {
	subject   Node
	predicate Node
	object    Node
	graph     Node
}

const (
	TypeNone = iota
	TypeBlank
	TypeIri
	TypeLiteral
)

func (st *Statement) String() string {
	var s, p, o string = "nil", "nil", "nil"
	
	if st.subject != nil   { s = st.subject.Encode() }
	if st.predicate != nil { p = st.predicate.Encode() }
	if st.object != nil    { o = st.object.Encode() }
	
	if st.graph == nil {
		return fmt.Sprintf("%s %s %s .", s, p, o)
	} else {
		return fmt.Sprintf("%s %s %s %s .", s, p, o, st.graph.Encode())
	}
}

func (st *Statement) SubjectType() int {
	switch st.subject.(type) {
		case *IriNode:   return TypeIri
		case *BlankNode: return TypeBlank
		default:         return TypeNone
	}
}

func (st *Statement) Subject() (iri string, subject_type int) {
	if iri, ok := st.subject.(*IriNode); ok {
		return iri.string, TypeIri
	} else if bn, ok := st.subject.(*BlankNode); ok {
		return bn.string, TypeBlank
	} else {
		return "", TypeNone
	}
}

func (st *Statement) Predicate() string {
	if iri, ok := st.predicate.(*IriNode); ok {
		return iri.string
	} else {
		return ""
	}
}

func (st *Statement) ObjectType() int {
	switch st.object.(type) {
		case *IriNode:     return TypeIri
		case *BlankNode:   return TypeBlank
		case *LiteralNode: return TypeLiteral
		default:           return TypeNone
	}
}

func (st *Statement) ObjectIri() (string, bool) {
	if iri, ok := st.object.(*IriNode); ok {
		return iri.string, true
	} else {
		return "", false
	}
}

func (st *Statement) ObjectBlank() (string, bool) {
	if bn, ok := st.object.(*BlankNode); ok {
		return bn.string, true
	} else {
		return "", false
	}
}

func (st *Statement) ObjectLiteral() (value, typ, lang string, ok bool) {
	if lit, ok := st.object.(*LiteralNode); ok {
		return lit.string, lit.Type.string, lit.Lang, true
	} else {
		return "", "", "", false
	}
}

func (st *Statement) HasGraph() bool {
	return st.graph != nil
}

func (st *Statement) Graph() (iri string, is_present bool) {
	if iri, ok := st.graph.(*IriNode); ok {
		return iri.string, true
	} else {
		return "", false
	}
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
