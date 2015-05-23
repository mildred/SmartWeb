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

type NQuadsBase struct {
	Statements []Statement
	last_node    Node
	subject      Node
	predicate    Node
	object       Node
	graph        Node
}

func (p *NQuadsBase) setSubject(){
	p.subject = p.last_node
	p.last_node = nil
}

func (p *NQuadsBase) setPredicate(){
	p.predicate = p.last_node
	p.last_node = nil
}

func (p *NQuadsBase) setObject(){
	p.object = p.last_node
	p.last_node = nil
}

func (p *NQuadsBase) setGraph(){
	p.graph = p.last_node
	p.last_node = nil
}

func (p *NQuadsBase) setStatement(){
	p.Statements = append(p.Statements, Statement{
		p.subject, p.predicate, p.object, p.graph,
	})
}

func (p *NQuadsBase) setLangTag(lang string){
	if s, ok := p.last_node.(*StringNode); ok {
		p.last_node = &LocStringNode { *s, lang }
	}
}

func (p *NQuadsBase) setIri(escaped_iri string){
	if s, ok := p.last_node.(*StringNode); ok {
		p.last_node = &TypedStringNode { *s, IriNode{ escaped_iri } }
	} else {
		p.last_node = &IriNode{ escaped_iri }
	}
}

func (p *NQuadsBase) setString(escaped_string string){
	p.last_node = &StringNode{escaped_string}
}

func (p *NQuadsBase) setBlank(blank_id string){
	p.last_node = &BlankNode{blank_id}
}