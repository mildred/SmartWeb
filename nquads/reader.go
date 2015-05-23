package nquads

import (
	"bufio"
	"io"
	"errors"
	l "log"
)

var ErrExpectedStatement = errors.New("Expected statement")
var ErrUnexpectedEscapeSequence = errors.New("Unexpected escape sequence")
var ErrUnexpectedCharInEscapeSequence = errors.New("Unexpected character in escape sequence")
var ErrUnsupported64bitsRune = errors.New("Unsupported 64 bits rune")
var ErrExpectedPredicate = errors.New("Expected predicate")
var ErrExpectedObject = errors.New("Expected object")
var ErrExpectedLiteralType = errors.New("Expected IRI for literal type after ^^")
var ErrInvalidCharacterInLanguageTag = errors.New("Invalid character in language tag")
var ErrExpectedFinalDot = errors.New("Expected final dot")

var EnableLogs = false

func log(args ...interface{}) {
	if EnableLogs {
		l.Println(args...)
	}
}

func logf(format string, args ...interface{}) {
	if EnableLogs {
		l.Printf(format, args...)
	}
}


type Reader struct {
	*bufio.Reader
	eofIsError bool
}

type ReadCloser struct {
	Reader
	io.Closer
}

func NewReader(r io.Reader) *Reader {
	return &Reader{bufio.NewReader(r), false}
}

func NewReadCloser(rc io.ReadCloser) *ReadCloser {
	return &ReadCloser{Reader{bufio.NewReader(rc), false}, rc}
}

func (r *Reader) ReadStatement() (*Statement, error) {
	st, err := r.readStatement()
	if err == io.EOF && r.eofIsError == false {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else if st == nil {
		return nil, ErrExpectedStatement
	} else {
		return st, nil
	}
}

func (p *Reader) readStatement() (*Statement, error) {
	var statement Statement
	var err error
	p.eofIsError = false

	statement.Subject, err = p.readSubject()
	if err != nil {
		return nil, err
	} else if statement.Subject == nil {
		return nil, nil
	}
	
	logf("Read subject %s", statement.String())
	
	p.eofIsError = true;
	
	statement.Predicate, err = p.readIri()
	if err != nil {
		return nil, err
	} else if statement.Predicate == nil {
		return nil, ErrExpectedPredicate
	}
	
	logf("Read predicate %s", statement.String())
	
	statement.Object, err = p.readObject()
	if err != nil {
		return nil, err
	} else if statement.Object == nil {
		return nil, ErrExpectedObject
	}

	logf("Read object %s", statement.String())
	
	err = p.readSpaces()
	if err != nil {
		return nil, err
	}
	
	_, err = p.Peek(1)
	if err != nil {
		return nil, err
	}
	
	statement.Graph, err = p.readIri()
	if err != nil {
		return nil, err
	}

	logf("Read graph %s", statement.String())
	
	err = p.readSpaces()
	if err != nil {
		return nil, err
	}
	
	c, err := p.ReadByte()
	if err != nil {
		return nil, err
	} else if c != '.' {
		return &statement, ErrExpectedFinalDot
	}
	
	p.eofIsError = false
	
	return &statement, nil
}

func (p *Reader) readSubject() (Node, error) {
	err := p.readSpaces()
	if err != nil {
		return nil, err
	}
	
	iri, err := p.readIri()
	if err != nil {
		return nil, err
	} else if iri != nil {
		return iri, nil
	}
	
	blank, err := p.readBlank()
	if err != nil {
		return nil, err
	} else {
		return blank, nil
	}
}

func (p *Reader) readObject() (Node, error) {
	node, err := p.readSubject()
	if err != nil {
		return nil, err
	} else if node != nil {
		return node, nil
	}
	
	return p.readLiteral()
}

func (p *Reader) readBlank() (Node, error) {
	err := p.readSpaces()
	if err != nil {
		return nil, err
	}
	
	prefix, err := p.Peek(2)
	if err != nil {
		return nil, err
	} else if string(prefix) != "_:" {
		return nil, nil
	}
	
	_, err = p.ReadBytes(':')
	if err != nil {
		return nil, err
	}
	
	var res string
	
	for {
		r, _, err := p.ReadRune()
		if err != nil {
			return nil, err
		}
		
		switch r {
			case '\t': fallthrough
			case '\r': fallthrough
			case '\n': fallthrough
			case ' ':
				p.UnreadRune()
				return &BlankNode{res}, nil
			default:
				res += string(r)
				continue
		}
	}
}

func (p *Reader) readLiteral() (Node, error) {
	err := p.readSpaces()
	if err != nil {
		return nil, err
	}
	
	r, _, err := p.ReadRune()
	if err != nil {
		return nil, err
	} else if r != '"' {
		p.UnreadRune()
		return nil, nil
	}
	
	var content string
	
	for {
		r, _, err = p.ReadRune()
		if err != nil {
			return nil, err
		} else if r == '"' {
			break
		} else if r == '\\' {
			r, err = p.readEscapeSeq()
			if err != nil {
				return nil, err
			}
		}
		content += string(r)
	}
	
	var stringNode LiteralNode = LiteralNode{content, "", IriNode{XsdString}}
	
	err = p.readSpaces()
	if err != nil {
		return nil, err
	}
	
	buf, err := p.Peek(2)
	if err != nil {
		return nil, err
	} else if string(buf) == "^^" {
		
		err = p.readSpaces()
		if err != nil {
			return nil, err
		}
		
		iri, err := p.readIri()
		if err != nil {
			return nil, err
		} else if iri, ok := iri.(*IriNode); ok {
			stringNode.Type = *iri
		} else {
			return nil, ErrExpectedLiteralType
		}
		
	} else if buf[0] == '@' {
		
		err = p.readSpaces()
		if err != nil {
			return nil, err
		}
		
		var langTag string
		
		loop:
		for {
			c, err := p.ReadByte()
			if err != nil {
				return nil, err
			}
			switch {
				default:
					if len(langTag) == 0 || langTag[len(langTag)-1] == '-' {
						return nil, ErrInvalidCharacterInLanguageTag
					}
					break loop
				case c >= 'A' && c <= 'Z':
				case c >= 'a' && c <= 'z':
				case c >= '0' && c <= '9':
				case c == '-':
					langTag += string(c)
					continue
			}
		}
		
		stringNode.Lang = langTag
		
	}
	
	return &stringNode, nil
}

func (p *Reader) readIri() (Node, error) {
	err := p.readSpaces()
	if err != nil {
		return nil, err
	}
	
	r, _, err := p.ReadRune()
	if err != nil {
		return nil, err
	} else if r != '<' {
		p.UnreadRune()
		return nil, nil
	}
	
	var uri []rune
	
	for {
		r, _, err = p.ReadRune()
		if err != nil {
			return nil, err
		} else if r == '>' {
			break
		} else if r == '\\' {
			r, err = p.readEscapeSeq()
			if err != nil {
				return nil, err
			}
		}
		uri = append(uri, r)
	}
	var s string = string(uri)
	logf("Read IRI <%s>", s)
	return &IriNode{s}, nil
}

// Assume \ has already been read
func (p *Reader) readEscapeSeq() (rune, error) {
	r, _, err := p.ReadRune()
	if err != nil {
		return 0, err
	}
	
	switch r {
		default:    return 0, ErrUnexpectedEscapeSequence
		case 't':	return 0x0009, nil
		case 'b':	return 0x0008, nil
		case 'n':	return 0x000A, nil
		case 'r':	return 0x000D, nil
		case 'f':	return 0x000C, nil
		case '"':	return 0x0022, nil
		case '\'':	return 0x0027, nil
		case '\\':	return 0x005C, nil
		case 'U':   fallthrough
		case 'u':
			var numDigits int
			if r == 'u' {
				numDigits = 4
			} else {
				numDigits = 8
			}
			var res uint64 = 0
			for i := 0; i < numDigits; i++ {
				c, err := p.ReadByte()
				if err != nil {
					return 0, err
				}
				var h byte
				if c >= '0' && c <= '9' {
					h = c - '0'
				} else if c >= 'a' && c <= 'f' {
					h = c - 'a' + 10
				} else if c >= 'A' || c <= 'F' {
					h = c - 'A' + 10
				} else {
					return 0, ErrUnexpectedCharInEscapeSequence
				}
				res = res | (uint64(h) << uint(numDigits-1-i)*8)
			}
			if res > 0xFFFFFFFF {
				return 0, ErrUnsupported64bitsRune
			} else {
				return rune(res), nil
			}
	}
}

func (p *Reader) readSpaces() error {
	inComment := false
	for {
		r, _, err := p.ReadRune()
		if err != nil {
			return err
		}
		
		if inComment {
			inComment = (r != '\r' && r != '\n')
			continue
		}
		
		switch r {
			default:
				p.UnreadRune()
				return nil
			case '#':
				inComment = true
				continue
			case ' ':  fallthrough
			case '\t': fallthrough
			case '\r': fallthrough
			case '\n':
				continue
		}
	}
}
