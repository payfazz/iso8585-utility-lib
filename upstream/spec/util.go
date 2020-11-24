package spec

import (
	"fmt"
	"io"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ReadOneMessage .
func ReadOneMessage(s Spec, r io.Reader, bufferIn []byte, bufferLenIn int) (msgOut Msg, msgRawOut []byte, bufferOut []byte, bufferLenOut int, err error) {
	var advance int
	var msg Msg

	buffer := bufferIn
	bufferLen := bufferLenIn
	needMore := 0

	for {
		for needMore > 0 || bufferLen == 0 {
			if len(buffer[bufferLen:]) <= needMore {
				newBuffer := make([]byte, max(cap(buffer)*2, 256))
				copy(newBuffer, buffer)
				buffer = newBuffer
			}

			readLen, err := r.Read(buffer[bufferLen:])
			if err != nil {
				return nil, nil, buffer, bufferLen, fmt.Errorf("read error: %s", err.Error())
			}

			bufferLen += readLen
			needMore -= readLen
		}

		advance, msg, needMore, err = s.MsgDecode(buffer[:bufferLen])
		if err != nil {
			return nil, nil, buffer, bufferLen, fmt.Errorf("decode error: %s", err.Error())
		}
		if needMore > 0 {
			continue
		}
		if msg == nil || advance == 0 {
			panic("spec error: invalid MsgDecode: ((msg == nil || advance == 0) && needMore <= 0 && err == nil)")
		}

		msgRaw := make([]byte, advance)
		copy(msgRaw, buffer[:advance])

		copy(buffer, buffer[advance:bufferLen])
		bufferLen -= advance
		needMore = 0

		return msg, msgRaw, buffer, bufferLen, nil
	}
}
