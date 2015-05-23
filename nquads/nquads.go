package nquads

import (
	"fmt"
)

type Statement struct {
	Subject   Node
	Predicate Node
	Object    Node
	Graph     Node
}

func (st *Statement) String() string {
	var s, p, o string = "nil", "nil", "nil"
	
	if st.Subject != nil   { s = st.Subject.Encode() }
	if st.Predicate != nil { p = st.Predicate.Encode() }
	if st.Object != nil    { o = st.Object.Encode() }
	
	if st.Graph == nil {
		return fmt.Sprintf("%s %s %s .", s, p, o)
	} else {
		return fmt.Sprintf("%s %s %s %s .", s, p, o, st.Graph.Encode())
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
