package main

import (
	"github.com/lxzan/gws"
	"net/http"
)

func main() {
	upgrader := gws.NewUpgrader(&Handler{}, &gws.ServerOption{
		CompressEnabled:  true,
		CheckUtf8Enabled: true,
	})
	http.HandleFunc("/connect", func(writer http.ResponseWriter, request *http.Request) {
		socket, err := upgrader.Upgrade(writer, request)
		if err != nil {
			return
		}
		go func() {
			socket.ReadLoop()
		}()
	})
	http.ListenAndServe(":8000", nil)
}

type Handler struct {
	gws.BuiltinEventHandler
}

func (c *Handler) OnPing(socket *gws.Conn, payload []byte) {
	_ = socket.WritePong(payload)
}

func (c *Handler) OnMessage(socket *gws.Conn, message *gws.Message) {
	defer message.Close()
	_ = socket.WriteMessage(message.Opcode, message.Bytes())
}
