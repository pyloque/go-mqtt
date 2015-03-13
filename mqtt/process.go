package mqtt

import (
	"log"
	"net"
)

func (room *Room) Kick() {
	if room.Started.isLocked() {
		return
	}
	if room.Started.Lock() {
		go func(room *Room) {
			for {
				select {
				case cm := <-room.Inbox:
					room.Handle(cm.ClientId, cm.Conn, cm.Message)
				default:
					break
				}
			}
			room.Started.UnLock()
		}(room)
	}
}

func (room *Room) Handle(clientId string, conn net.Conn, message Message) {
	log.Println(clientId, message)
	if message.hdr.Type == TypeDisconnect {
		conn.Close()
	}
}
