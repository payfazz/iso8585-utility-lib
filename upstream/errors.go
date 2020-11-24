package upstream

import "fmt"

// ErrServerClosed .
var ErrServerClosed = fmt.Errorf("Upstream already closed")

// ErrInvalidRequest .
type ErrInvalidRequest struct {
	Cause error
}

func (e *ErrInvalidRequest) Error() string {
	return e.Cause.Error()
}
