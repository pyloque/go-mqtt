package main

import (
	"log"
	"net"

	"github.com/pyloque/mqtt/codec"
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

func (room *Room) Handle(clientId string, conn net.Conn, message codec.Message) {
	log.Println(clientId, message)
	if message.Hdr.Type == codec.TypeDisconnect {
		conn.Close()
	}
}
