package vcread_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/koron-go/vcard/vcread"
)

func assertRead(t *testing.T, r *vcread.Reader, tokens ...vcread.Token) {
	t.Helper()
	for i, want := range tokens {
		got, err := r.Read()
		if err != nil {
			t.Fatalf("failed to read at #%d: %s", i, err)
		}
		if wantType, gotType := want.Type(), got.Type(); gotType != wantType {
			t.Errorf("unexpected token type: want=%v got=%v", wantType, gotType)
		}
		if d := cmp.Diff(want, got); d != "" {
			t.Errorf("unexpected token at #%d: -want +got\n:%s", i, d)
		}
	}
	last, err := r.Read()
	if err == nil {
		t.Errorf("unexpected success of read, expected io.EOF")
	} else if !errors.Is(err, io.EOF) {
		t.Errorf("unexpected failure of read, expected io.EOF: %s", err)
	}
	if last != nil {
		t.Errorf("read unexpected token, expected nil: %+v", last)
	}
}

func TestSimple(t *testing.T) {
	assertRead(t,
		vcread.New(strings.NewReader("BEGIN:VCARD\r\nVERSION:2.1\r\nEND:VCARD\r\n")),
		&vcread.NameToken{
			NameBytes: []byte("BEGIN"),
		},
		&vcread.ValueToken{
			ValueBytes: []byte("VCARD\r\n"),
			Continue:   false,
		},
		&vcread.NameToken{
			NameBytes: []byte("VERSION"),
		},
		&vcread.ValueToken{
			ValueBytes: []byte("2.1\r\n"),
			Continue:   false,
		},
		&vcread.NameToken{
			NameBytes: []byte("END"),
		},
		&vcread.ValueToken{
			ValueBytes: []byte("VCARD\r\n"),
			Continue:   false,
		},
	)
}

func TestFoldedLine(t *testing.T) {
	assertRead(t,
		vcread.New(strings.NewReader("BEGIN:VCARD\r\n VERSION:2.1\r\nEND:VCARD\r\n")),
		&vcread.NameToken{
			NameBytes: []byte("BEGIN"),
		},
		&vcread.ValueToken{
			ValueBytes: []byte("VCARD\r\n"),
			Continue:   true,
		},
		&vcread.ValueToken{
			ValueBytes: []byte("VERSION:2.1\r\n"),
			Continue:   false,
		},
		&vcread.NameToken{
			NameBytes: []byte("END"),
		},
		&vcread.ValueToken{
			ValueBytes: []byte("VCARD\r\n"),
			Continue:   false,
		},
	)
}

func TestParams(t *testing.T) {
	assertRead(t,
		vcread.New(strings.NewReader("FN;B;CHARSET=UTF-8:John Doe\r\n")),
		&vcread.NameToken{
			NameBytes: []byte("FN"),
		},
		&vcread.ParamToken{
			NameBytes: []byte("B"),
		},
		&vcread.ParamToken{
			NameBytes:  []byte("CHARSET"),
			ValueBytes: []byte("UTF-8"),
		},
		&vcread.ValueToken{
			ValueBytes: []byte("John Doe\r\n"),
			Continue:   false,
		},
	)
}

func TestIncompleteValue(t *testing.T) {
	assertRead(t,
		vcread.New(strings.NewReader("BEGIN:VCARD")),
		&vcread.NameToken{
			NameBytes: []byte("BEGIN"),
		},
		&vcread.ValueToken{
			ValueBytes: []byte("VCARD"),
			Continue:   false,
		},
	)
}

func TestErrIncompleteName(t *testing.T) {
	r := vcread.New(strings.NewReader("BEGIN"))
	tok, err := r.Read()
	if !errors.Is(err, vcread.ErrIncompleteName) {
		t.Errorf("unexpected error, expected vcread.ErrIncompleteName: %s", err)
	}
	if tok != nil {
		t.Errorf("read unexpected token, expected nil: %+v", tok)
	}
}
