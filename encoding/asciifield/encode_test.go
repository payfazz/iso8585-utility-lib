package asciifield_test

import (
	"bytes"
	"testing"

	"github.com/payfazz/iso8585-utility-lib/encoding/asciifield"
)

func TestEncodeFix1(t *testing.T) {
	field := asciifield.FixSize(4)
	decoded := "abcd"
	output, err := field.Encode(decoded)
	if err != nil {
		t.Fatalf("invalid err")
	}
	if bytes.Compare([]byte(decoded), output) != 0 {
		t.Fatalf("invalid encoded")
	}
}

func TestEncodeFix2(t *testing.T) {
	field := asciifield.FixSize(4)
	decoded := "abc\n"
	_, err := field.Encode(decoded)
	if err != asciifield.ErrInvalidCharset {
		t.Fatalf("invalid err")
	}
}

func TestEncodeFix3(t *testing.T) {
	field := asciifield.FixSize(4)
	decoded := "abc"
	_, err := field.Encode(decoded)
	if err != asciifield.ErrInvalidLength {
		t.Fatalf("invalid err")
	}
}

func TestEncodeLVar1(t *testing.T) {
	field := asciifield.LVar()
	decoded := "abc"
	encoded := []byte("3abc")
	output, err := field.Encode(decoded)
	if err != nil {
		t.Fatalf("invalid err")
	}
	if bytes.Compare(encoded, output) != 0 {
		t.Fatalf("invalid encoded")
	}
}

func TestEncodeLVar2(t *testing.T) {
	field := asciifield.LVar()
	decoded := "0123456789012"
	_, err := field.Encode(decoded)
	if err != asciifield.ErrInvalidLength {
		t.Fatalf("invalid err")
	}
}

func TestEncodeLLVar1(t *testing.T) {
	field := asciifield.LLVar()
	decoded := "abc"
	encoded := []byte("03abc")
	output, err := field.Encode(decoded)
	if err != nil {
		t.Fatalf("invalid err")
	}
	if bytes.Compare(encoded, output) != 0 {
		t.Fatalf("invalid encoded")
	}
}

func TestEncodeLLLVar1(t *testing.T) {
	field := asciifield.LLLVar()
	decoded := "abc"
	encoded := []byte("003abc")
	output, err := field.Encode(decoded)
	if err != nil {
		t.Fatalf("invalid err")
	}
	if bytes.Compare(encoded, output) != 0 {
		t.Fatalf("invalid encoded")
	}
}
