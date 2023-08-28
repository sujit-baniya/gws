package main

import (
	"log"
	"net/http"

	"github.com/lxzan/gws"
)

func main() {
	var app = gws.NewServer(new(Handler), nil)

	app.OnRequest = func(socket *gws.Conn, request *http.Request) {
		var channel = make(chan []byte, 8)
		var closer = make(chan struct{})
		socket.SessionStorage.Store("channel", channel)
		socket.SessionStorage.Store("closer", closer)
		go socket.ReadLoop()
		go func() {
			for {
				select {
				case p := <-channel:
					_ = socket.WriteMessage(gws.OpcodeText, p)
				case <-closer:
					return
				}
			}
		}()
	}

	log.Fatalf("%v", app.Run(":8000"))
}

type Handler struct {
	gws.BuiltinEventHandler
}

func (c *Handler) MustGet(ss gws.SessionStorage, key string) any {
	v, _ := ss.Load(key)
	return v
}

func (c *Handler) Send(socket *gws.Conn, payload []byte) {
	var channel = c.MustGet(socket.SessionStorage, "channel").(chan []byte)
	select {
	case channel <- payload:
	default:
		return
	}
}

func (c *Handler) OnClose(socket *gws.Conn, err error) {
	var closer = c.MustGet(socket.SessionStorage, "closer").(chan struct{})
	closer <- struct{}{}
}

func (c *Handler) OnMessage(socket *gws.Conn, message *gws.Message) {
	defer message.Close()
	_ = socket.WriteMessage(message.Opcode, message.Bytes())
}
