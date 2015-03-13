package main

import (
	"bufio"
	"log"
	"net"

	"github.com/pyloque/mqtt/codec"
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
	mqtt := codec.NewMQTTReader(bufio.NewReader(conn))
	suck := mqtt.ReadIter()
	message, err := suck()
	if err != nil {
		log.Println(err)
		return
	}
	if message.Hdr.Type != codec.TypeConnect {
		log.Println("first message must be a Connect type")
		return
	}
	clientId := message.Body.(*codec.Connect).ClientId
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
