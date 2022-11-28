package vcread

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/koron-go/vcard/internal/errio"
)

func assertReadErr(t *testing.T, r *Reader, wantErr string) {
	t.Helper()
	got, err := r.Read()
	if err == nil {
		t.Errorf("unexpected success of read, expected io.EOF")
	} else if gotErr := err.Error(); gotErr != wantErr {
		t.Errorf("unexpected error\nwant=%s\ngot=%s", wantErr, gotErr)
	}
	if got != nil {
		t.Errorf("read unexpected token, expected nil: %#v", got)
	}
}

func TestCoverageMode(t *testing.T) {
	r := New(strings.NewReader(""))
	r.mode = -1
	assertReadErr(t, r, "unknown parse mode: -1")
}

func TestCoverageEncoding(t *testing.T) {
	r := New(strings.NewReader("N:FOO"))
	tok1, err := r.Read()
	if err != nil {
		t.Fatalf("unexpected error at #0 read: %s", err)
	}
	if d := cmp.Diff(&NameToken{NameBytes: []byte("N")}, tok1); d != "" {
		t.Errorf("unexpected token at #%d: -want +got\n:%s", 0, d)
	}
	r.encoding = -1
	assertReadErr(t, r, "unknown encoding: -1")
}

func TestCoverageIOErr(t *testing.T) {
	const myErr = "this is dummy error"
	r := New(errio.NewReader(strings.NewReader("DUMMY:"), errors.New(myErr)))
	tok1, err := r.Read()
	if err != nil {
		t.Fatalf("unexpected error at #0 read: %s", err)
	}
	if d := cmp.Diff(&NameToken{NameBytes: []byte("DUMMY")}, tok1); d != "" {
		t.Errorf("unexpected token at #%d: -want +got\n:%s", 0, d)
	}
	assertReadErr(t, r, myErr)
}
