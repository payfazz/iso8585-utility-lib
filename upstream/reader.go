package upstream

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/payfazz/iso8585-utility-lib/upstream/spec"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (u *Upstream) reader(ctx context.Context, conn net.Conn, unprocessedRead []byte) error {
	buffer := make([]byte, max(1024, len(unprocessedRead)))
	copy(buffer, unprocessedRead)

	bufferLen := len(unprocessedRead)
	needMore := 0

	for {
		for needMore > 0 || bufferLen == 0 {
			if len(buffer[bufferLen:]) < needMore {
				newBuffer := make([]byte, cap(buffer)*2)
				copy(newBuffer, buffer)
				buffer = newBuffer
			}

			readLen, err := conn.Read(buffer[bufferLen:])
			if err != nil {
				return fmt.Errorf("read error: %s", err.Error())
			}

			bufferLen += readLen
			needMore -= readLen
		}

		advance, msg, needMoreDecode, err := u.spec.MsgDecode(buffer[:bufferLen])
		if err != nil {
			return fmt.Errorf("decode error: %s", err.Error())
		}
		if needMoreDecode > 0 {
			needMore = needMoreDecode
			continue
		}
		if msg == nil {
			panic("spec error: invalid MsgDecode: msg == nil && err == nil && needMore == 0")
		}

		msgRaw := make([]byte, advance)
		copy(msgRaw, buffer[:advance])

		copy(buffer, buffer[advance:bufferLen])
		bufferLen -= advance
		needMore = 0

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
