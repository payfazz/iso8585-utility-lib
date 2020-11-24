package asciifield_test

import (
	"testing"

	"github.com/payfazz/iso8585-utility-lib/encoding/asciifield"
)

func TestDecodeFix1(t *testing.T) {
	size := 4
	field := asciifield.FixSize(size)
	encoded := []byte("abcdef")
	decoded := "abcd"
	advance, output, _, err := field.Decode(encoded)
	if err != nil {
		t.Fatalf("invalid err")
	}
	if advance != size {
		t.Fatalf("invalid advance")
	}
	if output != decoded {
		t.Fatalf("invalid decoded")
	}
}

func TestDecodeFix2(t *testing.T) {
	size := 4
	field := asciifield.FixSize(size)
	encoded := []byte("ab")
	_, _, needMore, err := field.Decode(encoded)
	if err == nil && needMore > 0 {
		if needMore != 2 {
			t.Fatalf("invalid needMore")
		}
	} else {
		t.Fatalf("invalid err")
	}
}

func TestDecodeFix3(t *testing.T) {
	size := 4
	field := asciifield.FixSize(size)
	encoded := []byte("abc\n")
	_, _, _, err := field.Decode(encoded)
	if err != asciifield.ErrInvalidCharset {
		t.Fatalf("invalid err")
	}
}

func TestDecodeLLLVar1(t *testing.T) {
	field := asciifield.LLLVar()
	encoded := []byte("0")
	_, _, needMore, err := field.Decode(encoded)
	if err == nil && needMore > 0 {
		if needMore != 2 {
			t.Fatalf("invalid needMore")
		}
	} else {
		t.Fatalf("invalid err")
	}
}

func TestDecodeLLLVar2(t *testing.T) {
	field := asciifield.LLLVar()
	encoded := []byte("aa\n")
	_, _, _, err := field.Decode(encoded)
	if err != asciifield.ErrInvalidLength {
		t.Fatalf("invalid err")
	}
}

func TestDecodeLLLVar3(t *testing.T) {
	field := asciifield.LLLVar()
	encoded := []byte("aaa")
	_, _, _, err := field.Decode(encoded)
	if err != asciifield.ErrInvalidLength {
		t.Fatalf("invalid err")
	}
}

func TestDecodeLLLVar4(t *testing.T) {
	field := asciifield.LLLVar()
	encoded := []byte("003a")
	_, _, needMore, err := field.Decode(encoded)
	if err == nil && needMore > 0 {
		if needMore != 2 {
			t.Fatalf("invalid needMore")
		}
	} else {
		t.Fatalf("invalid err: %#v\n", err)
	}
}

func TestDecodeLLLVar5(t *testing.T) {
	field := asciifield.LLLVar()
	size := 6
	encoded := []byte("003aaabb")
	decoded := "aaa"
	advance, output, _, err := field.Decode(encoded)
	if err != nil {
		t.Fatalf("invalid err")
	}
	if advance != size {
		t.Fatalf("invalid advance")
	}
	if output != decoded {
		t.Fatalf("invalid decoded")
	}
}

func TestDecodeLLLVar6(t *testing.T) {
	field := asciifield.LLLVar()
	encoded := []byte("003aa\na")
	_, _, _, err := field.Decode(encoded)
	if err != asciifield.ErrInvalidCharset {
		t.Fatalf("invalid err")
	}
}
