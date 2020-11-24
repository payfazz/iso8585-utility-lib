package upstream

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/payfazz/iso8585-utility-lib/upstream/spec"
)

var netDialer = &net.Dialer{}
var defTime = time.Time{}

type submission struct {
	id string

	sendOnly bool
	send     struct {
		msg spec.Msg
		raw []byte
	}

	recv struct {
		msg spec.Msg
		raw []byte
	}

	err error

	doneCh chan struct{}
}

func newSubmission(id string, msg spec.Msg, msgRaw []byte, sendOnly bool) *submission {
	s := &submission{}
	s.id = id
	s.send.msg = msg
	s.send.raw = msgRaw
	s.sendOnly = sendOnly
	s.doneCh = make(chan struct{})
	return s
}

func (s *submission) isDone() bool {
	select {
	case <-s.doneCh:
		return true
	default:
		return false
	}
}

func (s *submission) setOk(msg spec.Msg, msgRaw []byte) {
	if s.isDone() {
		return
	}

	s.recv.msg = msg
	s.recv.raw = msgRaw

	close(s.doneCh)
}

func (s *submission) setErr(err error) {
	if s.isDone() {
		return
	}

	s.err = err
	close(s.doneCh)
}

func (u *Upstream) logInfo(format string, args ...interface{}) {
	if u.logger.Info != nil {
		u.logger.Info(fmt.Sprintf(format, args...))
	}
}

func (u *Upstream) logErr(format string, args ...interface{}) {
	if u.logger.Err != nil {
		u.logger.Err(fmt.Sprintf(format, args...))
	}
}

func (u *Upstream) isClosed() bool {
	return u.lifetimeCtx.Err() != nil
}

// simple net.Conn wrapper, that can only be closed once
type connCloserHelper struct {
	net.Conn
	closer sync.Once
	err    error
}

func (c *connCloserHelper) Close() error {
	c.closer.Do(func() { c.err = c.Conn.Close() })
	return c.err
}
