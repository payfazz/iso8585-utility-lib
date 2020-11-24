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

		go func() {
			u.logInfo("R: %s", msg.String())
			u.processRecvMsg(msg, msgRaw)
		}()
	}
}

func (u *Upstream) processRecvMsg(msg spec.Msg, msgRaw []byte) {
	ctx := u.lifetimeCtx

	if autoResp := u.spec.AutoResp(msg); autoResp != nil {
		u.sendAutoResp(ctx, autoResp)
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

func (u *Upstream) sendAutoResp(ctx context.Context, msg spec.Msg) {
	msgRaw, err := u.spec.MsgEncode(msg)
	if err != nil {
		panic("spec error: " + err.Error())
	}

	s := newSubmission(u.spec.MsgID(msg), msg, msgRaw, true)

	u.submission.data.lock.Lock()
	u.submission.data.data[s.id] = s
	u.submission.data.lock.Unlock()

	removeSubmission := func(err error) {
		u.submission.data.lock.Lock()
		delete(u.submission.data.data, s.id)
		u.submission.data.lock.Unlock()
		s.setErr(err)
	}

	select {
	case <-ctx.Done():
		removeSubmission(ctx.Err())
		return
	case u.submission.priorityNotify <- s:
	}
}
