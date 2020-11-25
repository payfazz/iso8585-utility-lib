package upstream

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/payfazz/iso8585-utility-lib/upstream/spec"
	"github.com/payfazz/mainutil/maintls"
)

// Upstream .
type Upstream struct {
	lifetimeCtx       context.Context
	cancelLifetimeCtx context.CancelFunc
	wait              sync.WaitGroup

	target string
	spec   spec.Spec
	logger struct {
		Info func(string)
		Err  func(string)
	}

	proxy struct {
		endpoint  *url.URL
		tlsConfig *tls.Config
		caSum     string
	}

	submission struct {
		data struct {
			lock sync.RWMutex
			data map[string]*submission
		}

		priorityNotify chan *submission
		notify         chan *submission
	}

	lastActive int64
}

// Build .
func (b Builder) Build() (*Upstream, error) {
	u := b.inner
	if u.target == "" {
		return nil, fmt.Errorf("invalid target")
	}
	if u.spec == nil {
		return nil, fmt.Errorf("invalid spec")
	}
	if u.proxy.endpoint != nil {
		u.proxy.endpoint.Scheme = strings.ToLower(u.proxy.endpoint.Scheme)
		switch u.proxy.endpoint.Scheme {
		case "https":
			tlsConfig := maintls.TLSConfig()
			tlsConfig.ServerName = u.proxy.endpoint.Hostname()
			if u.proxy.caSum != "" {
				maintls.SetStaticPeerVerification(tlsConfig, true, u.proxy.caSum)
			}
			u.proxy.tlsConfig = tlsConfig
			if u.proxy.endpoint.Port() == "" {
				u.proxy.endpoint.Host = u.proxy.endpoint.Hostname() + ":443"
			}
		case "http":
			if u.proxy.endpoint.Port() == "" {
				u.proxy.endpoint.Host = u.proxy.endpoint.Hostname() + ":80"
			}
		default:
			return nil, fmt.Errorf("invalid proxy scheme: %s", u.proxy.endpoint.Scheme)
		}
	}

	u.lifetimeCtx, u.cancelLifetimeCtx = context.WithCancel(context.Background())

	u.submission.data.data = make(map[string]*submission)
	u.submission.priorityNotify = make(chan *submission)
	u.submission.notify = make(chan *submission)

	u.wait.Add(1)
	go func() {
		defer u.wait.Done()
		u.main()
	}()

	return u, nil
}

// Close .
func (u *Upstream) Close() error {
	if u.isClosed() {
		return nil
	}

	u.cancelLifetimeCtx()
	u.wait.Wait()

	return nil
}

func (u *Upstream) main() {
	processUpstreamConnection := func() error {
		var wait sync.WaitGroup
		defer wait.Wait()

		ctx, cancelCtx := context.WithCancel(u.lifetimeCtx)
		defer cancelCtx()

		conn, unprocessed, err := u.dial(ctx)
		if err != nil {
			return err
		}

		defer func() {
			u.logInfo("closing connection")
			conn.Close()
		}()

		u.logInfo("connected")

		atomic.StoreInt64(&u.lastActive, time.Now().Unix())

		errCh := make(chan error, 1)
		passErr := func(err error) {
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}

		// r-lock submission data, so no new submission is submited
		u.submission.data.lock.RLock()

		// snapshot submission
		var snapshot []*submission
		for _, v := range u.submission.data.data {
			snapshot = append(snapshot, v)
		}

		prioritized := make(map[*submission]struct{})

		// flush notification when holding r-lock to submission data,
		// to avoid duplicate notification on new writer
		// NOTE: need to wrap in function call, so RUnlock() after it always called
		func() {
			for {
				select {
				case <-u.submission.notify:
				case s := <-u.submission.priorityNotify:
					prioritized[s] = struct{}{}
				case <-ctx.Done():
					return
				default:
					return
				}
			}
		}()

		// unlock submission data,
		// snapshoted submission is now clear from pending notification
		u.submission.data.lock.RUnlock()

		wait.Add(1)
		go func() {
			defer wait.Done()

			// renotify all snapshoted submission
			for _, v := range snapshot {
				if _, ok := prioritized[v]; ok {
					select {
					case <-ctx.Done():
						return
					case u.submission.priorityNotify <- v:
					}
				} else {
					select {
					case <-ctx.Done():
						return
					case u.submission.notify <- v:
					}
				}
			}
		}()

		wait.Add(1)
		go func() {
			defer wait.Done()
			passErr(u.reader(ctx, conn, unprocessed))
		}()

		wait.Add(1)
		go func() {
			defer wait.Done()
			passErr(u.writer(ctx, conn))
		}()

		wait.Add(1)
		go func() {
			defer wait.Done()
			passErr(u.pinger(ctx))
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		}
	}

	for {
		err := processUpstreamConnection()

		if u.isClosed() {
			return
		}

		if err != nil {
			if err == context.DeadlineExceeded {
				err = fmt.Errorf("timeout")
			}
			u.logErr("connection error (reconnect in 0.5s): %s", err.Error())
			select {
			case <-u.lifetimeCtx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
}

func (u *Upstream) dial(ctx context.Context) (net.Conn, []byte, error) {
	dialCtx, cancelDialCtx := context.WithTimeout(ctx, 5*time.Second)
	defer cancelDialCtx()

	dialTarget := u.target
	if u.proxy.endpoint != nil {
		dialTarget = u.proxy.endpoint.Host
	}
	conn, err := netDialer.DialContext(dialCtx, "tcp", dialTarget)
	if err != nil {
		return nil, nil, err
	}
	conn = &connCloserHelper{Conn: conn}

	var unprocessedRead []byte

	if u.proxy.endpoint != nil {
		if u.proxy.endpoint.Scheme == "https" {
			conn = tls.Client(conn, u.proxy.tlsConfig)
		}

		req := &http.Request{
			Method: http.MethodConnect,
			URL:    &url.URL{Opaque: u.target},
			Header: http.Header{
				"Host": []string{u.target},
			},
		}
		user := u.proxy.endpoint.User.Username()
		pass, _ := u.proxy.endpoint.User.Password()
		if user != "" || pass != "" {
			req.Header["Proxy-Authorization"] = []string{
				"Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass)),
			}
		}

		// background goroutine to force close connection,
		// because req.write and http.ReadResponse doesn't have timeout functionality
		// dialState:
		// 0 => not done, waiting
		// 1 => done
		// 2 => not done, timed up, will be closed
		dialState := int32(0)
		go func() {
			<-dialCtx.Done()
			if atomic.CompareAndSwapInt32(&dialState, 0, 2) {
				conn.Close()
			}
		}()

		if err := req.Write(conn); err != nil {
			conn.Close()
			if dialCtx.Err() != nil {
				err = fmt.Errorf("timeout")
			}
			return nil, nil, fmt.Errorf("failed to write http proxy CONNECT request: %s", err.Error())
		}

		buff := bufio.NewReader(conn)
		resp, err := http.ReadResponse(buff, req)
		if err != nil {
			conn.Close()
			if dialCtx.Err() != nil {
				err = fmt.Errorf("timeout")
			}
			return nil, nil, fmt.Errorf("failed to read http proxy CONNECT request: %s", err.Error())
		}

		if resp.StatusCode != 200 {
			conn.Close()

			if squidError := resp.Header.Get("X-Squid-Error"); squidError != "" {
				return nil, nil, fmt.Errorf("http proxy returning %d: %v", resp.StatusCode, squidError)
			}

			return nil, nil, fmt.Errorf("http proxy returning %d", resp.StatusCode)
		}

		if !atomic.CompareAndSwapInt32(&dialState, 0, 1) {
			return nil, nil, fmt.Errorf("http proxy CONNECT failed: timeout")
		}

		if buff.Buffered() > 0 {
			unprocessedRead, _ = buff.Peek(buff.Buffered())
		}
	}

	onNewConnCtx, cancelonNewConnCtx := context.WithTimeout(ctx, 10*time.Second)
	defer cancelonNewConnCtx()

	unprocessedRead, err = u.spec.OnNewConn(onNewConnCtx, conn, unprocessedRead)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("OnNewConn failed: %s", err.Error())
	}

	return conn, unprocessedRead, nil
}

// Process .
func (u *Upstream) Process(ctx context.Context, msg spec.Msg) (res spec.Msg, err error) {
	msgRaw, err := u.spec.MsgEncode(msg)
	if err != nil {
		return nil, &ErrInvalidRequest{Cause: err}
	}

	s := newSubmission(u.spec.MsgID(msg), msg, msgRaw, false)

	u.submission.data.lock.Lock()
	_, duplicate := u.submission.data.data[s.id]
	if !duplicate {
		u.submission.data.data[s.id] = s
	}
	u.submission.data.lock.Unlock()

	if duplicate {
		return nil, &ErrInvalidRequest{Cause: fmt.Errorf("duplicate ongoing request")}
	}

	removeSubmission := func(err error) {
		s.setErr(err)
		u.submission.data.lock.Lock()
		delete(u.submission.data.data, s.id)
		u.submission.data.lock.Unlock()
	}

	select {
	case <-u.lifetimeCtx.Done():
		removeSubmission(ErrServerClosed)
		return nil, ErrServerClosed

	case <-ctx.Done():
		removeSubmission(ctx.Err())
		return nil, ctx.Err()

	case u.submission.notify <- s:
	}

	select {
	case <-u.lifetimeCtx.Done():
		removeSubmission(ErrServerClosed)
		return nil, ErrServerClosed

	case <-ctx.Done():
		removeSubmission(ctx.Err())
		return nil, ctx.Err()

	case <-s.doneCh:
		return s.recv.msg, s.err
	}
}
