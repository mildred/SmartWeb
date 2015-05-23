package nquads

import (
	"io"
	"errors"
	"strings"
	"fmt"
)

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