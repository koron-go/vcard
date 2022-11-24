package vcread

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

type Type int

const (
	NameType Type = iota
	ParamType
	ValueType
)

type Reader struct {
	br *bufio.Reader

	mode Type
	pend []byte
}

func New(r io.Reader) *Reader {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &Reader{
		br:   br,
		mode: NameType,
	}
}

type Token interface {
	Type() Type
}

type NameToken struct {
	NameBytes []byte
}

func (nt *NameToken) Type() Type {
	return NameType
}

type ParamToken struct {
	NameBytes  []byte
	ValueBytes []byte
}

func (nt *ParamToken) Type() Type {
	return ParamType
}

var _ Token = (*ParamToken)(nil)

type ValueToken struct {
	ValueBytes []byte
	Continue   bool
}

func (nt *ValueToken) Type() Type {
	return ValueType
}

var _ Token = (*ValueToken)(nil)

func (r *Reader) Read() (Token, error) {
	switch r.mode {
	case NameType:
		return r.readName()
	case ParamType:
		return r.readParam()
	case ValueType:
		return r.readValue()
	}
	return nil, fmt.Errorf("unknown parse mode: %d", r.mode)
}

var ErrIncompleteName = errors.New("incomplete name, missing a colon")

func (r *Reader) readName() (*NameToken, error) {
	b, err := r.br.ReadBytes(':')
	if err != nil {
		if errors.Is(err, io.EOF) && len(b) > 0 {
			return nil, ErrIncompleteName
		}
		return nil, err
	}
	// remove delimiter ':' for value
	b = b[:len(b)-1]

	// find delimiter ';' for parameters
	x := bytes.IndexByte(b, ';')
	// no parameters
	if x == -1 {
		r.mode = ValueType
		return &NameToken{NameBytes: b}, nil
	}

	bn, bp := b[:x], b[x+1:]
	r.mode = ParamType
	r.pend = bp
	return &NameToken{NameBytes: bn}, nil
}

func (r *Reader) readParam() (*ParamToken, error) {
	var curr []byte
	// find delimiter ';' for next parameters
	x := bytes.IndexByte(r.pend, ';')
	if x != -1 {
		// has a delimiter ';' for next parameters
		curr = r.pend[:x]
		r.pend = r.pend[x+1:]
	} else {
		// no more parameters
		curr = r.pend
		r.pend = nil
		r.mode = ValueType
	}

	y := bytes.IndexByte(curr, '=')
	if y == -1 {
		// parameter without values
		return &ParamToken{NameBytes: curr}, nil
	}
	name, value := curr[:y], curr[y+1:]
	// FIXME: parse special cases (ex. ENCODING=QUOTED-PRINTABLE)
	// parameter with a value
	return &ParamToken{NameBytes: name, ValueBytes: value}, nil
}

func (r *Reader) readValue() (*ValueToken, error) {
	b, err := r.br.ReadBytes('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			r.mode = NameType
			return &ValueToken{
				ValueBytes: b,
				Continue:   false,
			}, nil
		}
		return nil, err
	}
	// read a next byte to check is it a white space?
	next, err := r.br.ReadByte()
	if err != nil {
		if errors.Is(err, io.EOF) {
			r.mode = NameType
			return &ValueToken{
				ValueBytes: b,
				Continue:   false,
			}, nil
		}
		return nil, err
	}
	// next line folded.
	if next == ' ' || next == '\t' {
		return &ValueToken{
			ValueBytes: b,
			Continue:   true,
		}, nil
	}
	// no next line.
	r.br.UnreadByte()
	r.mode = NameType
	return &ValueToken{
		ValueBytes: b,
		Continue:   false,
	}, nil
}
