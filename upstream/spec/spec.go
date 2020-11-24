package spec

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"
)

// Spec .
type Spec interface {
	OnNewConn(ctx context.Context, conn net.Conn, readed []byte) (unprocessedData []byte, err error)

	MsgEncode(decoded Msg) (encoded []byte, err error)
	MsgDecode(encoded []byte) (advance int, decoded Msg, needMore int, err error)
	MsgID(msg Msg) (id string)

	AutoResp(req Msg) (resp Msg)

	GetPingMsg() (ping Msg, duration time.Duration)
}

// Msg .
type Msg map[int]string

// String .
func (m Msg) String() string {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var b strings.Builder
	b.WriteByte('[')
	first := true
	for _, k := range keys {
		if first {
			first = false
		} else {
			b.WriteByte(' ')
		}
		v := m[k]
		if len(v) > 10 {
			v = v[:10] + "..."
		}
		b.WriteString(fmt.Sprintf("%d:%s", k, v))
	}
	b.WriteByte(']')
	return b.String()
}

// Clone .
func (m Msg) Clone() Msg {
	r := make(Msg)
	for k, v := range m {
		r[k] = v
	}
	return r
}
