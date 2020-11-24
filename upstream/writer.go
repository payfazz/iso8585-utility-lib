package upstream

import (
	"context"
	"fmt"
	"net"
)

func (u *Upstream) writer(ctx context.Context, conn net.Conn) error {
	write := func(s *submission) error {
		if s.isDone() {
			return nil
		}

		u.submission.data.lock.RLock()
		_, ok := u.submission.data.data[s.id]
		u.submission.data.lock.RUnlock()
		if !ok {
			return nil
		}

		u.logInfo("W: %s", s.send.msg)

		_, err := conn.Write(s.send.raw)

		if s.sendOnly {
			markSubmissionComplete := func() {
				u.submission.data.lock.Lock()
				delete(u.submission.data.data, s.id)
				u.submission.data.lock.Unlock()

				s.setOk(nil, nil)
			}
			go markSubmissionComplete()
		}

		if err != nil {
			return fmt.Errorf("write error: %s", err.Error())
		}

		return nil
	}

	for {
		select {
		case s := <-u.submission.priorityNotify:
			if err := write(s); err != nil {
				return err
			}
		default:
			select {
			case <-ctx.Done():
				return ctx.Err()
			case s := <-u.submission.notify:
				if err := write(s); err != nil {
					return err
				}
			case s := <-u.submission.priorityNotify:
				if err := write(s); err != nil {
					return err
				}
			}
		}
	}
}
