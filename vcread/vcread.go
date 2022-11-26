/*
Package vcread provides VCard reader.
*/
package vcread

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

type Type int

const (
	NameType Type = iota
	ParamType
	ValueType
)

type mode int

const (
	nameMode mode = iota
	paramMode
	valueMode
)

type encoding int

const (
	rawEnc encoding = iota
	qpEnc
	b64Enc
)

type Reader struct {
	br *bufio.Reader

	mode mode

	paramData []byte
	encoding  encoding
}

func New(r io.Reader) *Reader {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &Reader{
		br:   br,
		mode: nameMode,
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
	case nameMode:
		return r.readName()
	case paramMode:
		return r.readParam()
	case valueMode:
		return r.readValue()
	}
	return nil, fmt.Errorf("unknown parse mode: %d", r.mode)
}

var ErrIncompleteName = errors.New("incomplete name, missing a colon")

func (r *Reader) readName() (Token, error) {
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
		r.mode = valueMode
		r.paramData = nil
		r.encoding = rawEnc
		return &NameToken{NameBytes: b}, nil
	}

	bn, bp := b[:x], b[x+1:]
	r.mode = paramMode
	r.paramData = bp
	r.encoding = rawEnc
	return &NameToken{NameBytes: bn}, nil
}

func (r *Reader) readParam() (Token, error) {
	var curr []byte
	// find delimiter ';' for next parameters
	x := bytes.IndexByte(r.paramData, ';')
	if x != -1 {
		// has a delimiter ';' for next parameters
		curr = r.paramData[:x]
		r.paramData = r.paramData[x+1:]
	} else {
		// no more parameters
		curr = r.paramData
		r.paramData = nil
		r.mode = valueMode
	}

	y := bytes.IndexByte(curr, '=')
	if y == -1 {
		// parameter without values
		return &ParamToken{NameBytes: curr}, nil
	}
	name, value := curr[:y], curr[y+1:]
	// parse special cases (ex. ENCODING=QUOTED-PRINTABLE)
	if strings.ToUpper(string(name)) == "ENCODING" {
		encname := string(value)
		switch strings.ToUpper(encname) {
		case "7BIT", "8BIT":
			r.encoding = rawEnc
		case "QUOTED-PRINTABLE":
			r.encoding = qpEnc
		case "B", "BASE64":
			r.encoding = b64Enc
		default:
			return nil, fmt.Errorf("unknown encoding: %s", encname)
		}
	}
	// parameter with a value
	return &ParamToken{NameBytes: name, ValueBytes: value}, nil
}

func (r *Reader) readValue() (Token, error) {
	switch r.encoding {
	case rawEnc:
		return r.readValueRaw()
	case qpEnc:
		return r.readValueQP()
	case b64Enc:
		return r.readValueB64()
	default:
		return nil, fmt.Errorf("unknown encoding: %v", r.encoding)
	}
}

func (r *Reader) readValueRaw() (Token, error) {
	b, err := r.br.ReadBytes('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			r.mode = nameMode
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
			r.mode = nameMode
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
	r.mode = nameMode
	return &ValueToken{
		ValueBytes: b,
		Continue:   false,
	}, nil
}

func (r *Reader) readValueQP() (Token, error) {
	// TODO: read QUOTED-PRINTABLE encoded value.
	return r.readValueRaw()
}

func (r *Reader) readValueB64() (Token, error) {
	// TODO: read BASE64 encoded value.
	return r.readValueRaw()
}
