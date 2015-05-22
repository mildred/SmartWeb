package nquads

import (
	"io"
	"errors"
	"fmt"
	"strings"
)

type Node interface {}

type Statement struct {
	Subject   Node
	Predicate Node
	Object    Node
	Graph     Node
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

var ErrInvalidSubject   = errors.New("Invalid subject")
var ErrInvalidObject    = errors.New("Invalid object")

type NQuadWriter struct {
	io.Writer
}

func (w NQuadWriter) WriteEmptyLine() error {
	_, err := w.Write([]byte("\n"))
	return err
}

func (w NQuadWriter) WriteComment(comment string) error {
	_, err := w.Write([]byte(fmt.Sprintf("#%s\n", strings.Replace(comment, "\n", "\n#", -1))))
	return err
}

func (w NQuadWriter) WriteQuad(subject interface{}, predicate string, object interface{}, graph string) error {
	s := EncodeIriOrBlank(subject)
	p := EncodeIri(predicate)
	o := EncodeAny(object)
	g := EncodeIri(graph)
	if s == "" { return ErrInvalidSubject }
	if o == "" { return ErrInvalidObject }
	_, err := w.Write([]byte(fmt.Sprintf("%s %s %s %s .\n", s, p, o, g)))
	return err
}

func (w NQuadWriter) WriteQuadIri(subject interface{}, predicate, object, graph string) error {
	s := EncodeIriOrBlank(subject)
	p := EncodeIri(predicate)
	o := EncodeIri(object)
	g := EncodeIri(graph)
	if s == "" { return ErrInvalidSubject }
	_, err := w.Write([]byte(fmt.Sprintf("%s %s %s %s .\n", s, p, o, g)))
	return err
}

func (w NQuadWriter) WriteTriple(subject interface{}, predicate string, object interface{}) error {
	s := EncodeIriOrBlank(subject)
	p := EncodeIri(predicate)
	o := EncodeAny(object)
	if s == "" { return ErrInvalidSubject }
	if o == "" { return ErrInvalidObject }
	_, err := w.Write([]byte(fmt.Sprintf("%s %s %s .\n", s, p, o)))
	return err
}

func (w NQuadWriter) WriteTripleIri(subject interface{}, predicate string, object string) error {
	s := EncodeIriOrBlank(subject)
	p := EncodeIri(predicate)
	o := EncodeIri(object)
	if s == "" { return ErrInvalidSubject }
	_, err := w.Write([]byte(fmt.Sprintf("%s %s %s .\n", s, p, o)))
	return err
}