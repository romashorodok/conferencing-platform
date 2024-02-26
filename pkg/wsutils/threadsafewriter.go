package wsutils

import (
	"sync"

	"github.com/gorilla/websocket"
)

type ThreadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

func (t *ThreadSafeWriter) WriteJSON(val interface{}) error {
	t.Lock()
	defer t.Unlock()

	return t.Conn.WriteJSON(val)
}

func (t *ThreadSafeWriter) Close() error {
	return t.Conn.Close()
}

func (t *ThreadSafeWriter) ReadJSON(val any) error {
	return t.Conn.ReadJSON(val)
}

func NewThreadSafeWriter(conn *websocket.Conn) *ThreadSafeWriter {
	return &ThreadSafeWriter{
		Conn: conn,
	}
}
