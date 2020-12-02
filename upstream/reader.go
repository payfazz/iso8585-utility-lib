package upstream

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/payfazz/iso8585-utility-lib/upstream/spec"
)

func (u *Upstream) reader(ctx context.Context, conn net.Conn, unprocessedRead []byte) error {
	var msg spec.Msg
	var msgRaw []byte
	var err error

	buffer := unprocessedRead
	bufferLen := len(unprocessedRead)

	for {
		msg, msgRaw, buffer, bufferLen, err = spec.ReadOneMessage(u.spec, conn, buffer, bufferLen)
		if err != nil {
			return err
		}

		atomic.StoreInt64(&u.lastActive, time.Now().Unix())

		msg := msg
		msgRaw := msgRaw
		go func() {
			u.logInfo("R: %s", msg.String())
			u.processRecvMsg(msg, msgRaw)
		}()
	}
}

func (u *Upstream) processRecvMsg(msg spec.Msg, msgRaw []byte) {
	if autoResp := u.spec.AutoResp(msg); autoResp != nil {
		u.sendAutoResp(autoResp)
		return
	}

	id := u.spec.MsgID(msg)

	u.submission.data.lock.Lock()
	s := u.submission.data.data[id]
	delete(u.submission.data.data, id)
	u.submission.data.lock.Unlock()

	if s != nil {
		s.setOk(msg, msgRaw)
	}
}

func (u *Upstream) sendAutoResp(msg spec.Msg) {
	msgRaw, err := u.spec.MsgEncode(msg)
	if err != nil {
		panic("spec error: MsgEncode auto resp:" + err.Error())
	}

	s := newSubmission(u.spec.MsgID(msg), msg, msgRaw, true)

	u.submission.data.lock.Lock()
	u.submission.data.data[s.id] = s
	u.submission.data.lock.Unlock()

	removeSubmission := func(err error) {
		s.setErr(err)
		u.submission.data.lock.Lock()
		delete(u.submission.data.data, s.id)
		u.submission.data.lock.Unlock()
	}

	select {
	case <-u.lifetimeCtx.Done():
		removeSubmission(u.lifetimeCtx.Err())
		return
	case u.submission.priorityNotify <- s:
	}
}
