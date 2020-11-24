package asciifield

import (
	"fmt"
)

// Field .
type Field struct {
	fixSize int
	varSize int
}

// FixSize .
func FixSize(size int) Field {
	return Field{
		fixSize: size,
		varSize: 0,
	}
}

func varSize(size int) Field {
	return Field{
		fixSize: 0,
		varSize: size,
	}
}

// LVar .
func LVar() Field {
	return varSize(1)
}

// LLVar .
func LLVar() Field {
	return varSize(2)
}

// LLLVar .
func LLLVar() Field {
	return varSize(3)
}

// Encode .
func (e *Field) Encode(decoded string) (encoded []byte, err error) {
	if !validCharset(decoded) {
		return nil, ErrInvalidCharset
	}

	decodedBytes := []byte(decoded)

	if e.fixSize > 0 {
		if e.fixSize != len(decodedBytes) {
			return nil, ErrInvalidLength
		}
		return decodedBytes, nil
	}

	if e.varSize > 0 {
		maxLength := tenPow(e.varSize) - 1
		if len(decodedBytes) > maxLength {
			return nil, ErrInvalidLength
		}
		ret := []byte(fmt.Sprintf(fmt.Sprintf("%%0%dd", e.varSize), len(decodedBytes)))
		ret = append(ret, decodedBytes...)
		return ret, nil
	}

	panic("dead code: fixSize and varSize cannot be both 0")
}

// Decode .
func (e *Field) Decode(encoded []byte) (advance int, decoded string, needMore int, err error) {
	if e.fixSize > 0 {
		if len(encoded) < e.fixSize {
			return 0, "", e.fixSize - len(encoded), nil
		}
		decoded = string(encoded[:e.fixSize])
		if !validCharset(decoded) {
			return 0, "", 0, ErrInvalidCharset
		}
		return e.fixSize, decoded, 0, nil
	}

	if e.varSize > 0 {
		if len(encoded) < e.varSize {
			return 0, "", e.varSize - len(encoded), nil
		}

		decodedLen := int(0)
		decodedLenStr := string(encoded[:e.varSize])
		if !validDecimal(decodedLenStr) {
			return 0, "", 0, ErrInvalidLength
		}
		fmt.Sscanf(decodedLenStr, "%d", &decodedLen)

		encoded = encoded[e.varSize:]

		if len(encoded) < decodedLen {
			return 0, "", decodedLen - len(encoded), nil
		}

		decoded = string(encoded[:decodedLen])
		if !validCharset(decoded) {
			return 0, "", 0, ErrInvalidCharset
		}

		return decodedLen + e.varSize, decoded, 0, nil
	}

	panic("dead code: fixSize and varSize cannot be both 0")
}

func validCharset(s string) bool {
	for _, x := range s {
		if !(0x20 <= x && x <= 0x7E) {
			return false
		}
	}
	return true
}

func validDecimal(s string) bool {
	for _, x := range s {
		if !('0' <= x && x <= '9') {
			return false
		}
	}
	return true
}

func tenPow(p int) int {
	ret := 1
	for i := 0; i < p; i++ {
		ret *= 10
	}
	return ret
}
