package upstream

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

const minPingDuration = 5 * time.Second

func (u *Upstream) pinger(ctx context.Context) error {
	for {
		msg, duration := u.spec.GetPingMsg()
		if duration == 0 {
			return nil
		}

		if duration < minPingDuration {
			duration = minPingDuration
		}

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
			return ctx.Err()
		case u.submission.priorityNotify <- s:
		}

		select {
		case <-ctx.Done():
			removeSubmission(ctx.Err())
			return ctx.Err()
		case <-time.After(duration):
			removeSubmission(context.DeadlineExceeded)
		}

		if (time.Now().Unix() - atomic.LoadInt64(&u.lastActive)) > int64(duration.Seconds()) {
			return fmt.Errorf("inactive for %s", duration.String())
		}
	}
}
