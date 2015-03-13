package mqtt

import (
	"bufio"
	"log"
	"net"
)

func Start(protocol string, addr string) {
	l, err := net.Listen(protocol, addr)
	if err != nil {
		log.Fatalln(err)
	}
	defer l.Close()
	log.Println("Server Started")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go func(conn net.Conn) {
			Service(conn)
		}(conn)
	}
}

func Service(conn net.Conn) {
	defer conn.Close()
	room := GetDistrict().SelectRoom(DefaultRoomKey)
	mqtt := NewMQTTReader(bufio.NewReader(conn))
	suck := mqtt.ReadIter()
	message, err := suck()
	if err != nil {
		log.Println(err)
		return
	}
	if message.hdr.Type != TypeConnect {
		log.Println("first message must be a Connect type")
		return
	}
	clientId := message.body.(*Connect).ClientId
	room.Inbox <- ChannelMessage{clientId, conn, message}
	room.Kick()
	for {
		message, err = suck()
		if err != nil {
			log.Println(err)
			break
		}
		room.Inbox <- ChannelMessage{clientId, conn, message}
		room.Kick()
	}
}
