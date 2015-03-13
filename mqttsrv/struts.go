package main

import (
	"hash/adler32"
	"net"

	"github.com/pyloque/mqtt/codec"
)

var DefaultRoomKey = "default_room"

type District struct {
	Buildings []*Building
}

func NewDistrict(size int) *District {
	buildings := make([]*Building, size)
	for i := 0; i < len(buildings); i++ {
		buildings[i] = NewBuilding()
	}
	return &District{buildings}
}

func (self *District) GetBuilding(roomKey string) *Building {
	hashkey := adler32.Checksum([]byte(roomKey))
	buildings := self.Buildings
	return buildings[int(hashkey)%len(buildings)]
}

func (self *District) SelectRoom(roomKey string) *Room {
	building := self.GetBuilding(roomKey)
	return building.SelectRoom(roomKey)
}

type Building struct {
	Rooms map[string]*Room
}

func NewBuilding() *Building {
	return &Building{make(map[string]*Room)}
}

func (self *Building) GetRoom(roomKey string) *Room {
	return self.Rooms[roomKey]
}

func (self *Building) AddRoom(roomKey string) *Room {
	room := NewRoom()
	self.Rooms[roomKey] = room
	return room
}

func (self *Building) SelectRoom(roomKey string) *Room {
	room := self.GetRoom(roomKey)
	if room == nil {
		room = self.AddRoom(roomKey)
	}
	return room
}

type Room struct {
	Started     *Atomic
	Connections map[string]*Connection
	Inbox       chan ChannelMessage
}

func NewRoom() *Room {
	return &Room{&Atomic{0}, make(map[string]*Connection), make(chan ChannelMessage, 100)}
}

func (self *Room) AddConnection(clientId string, conn net.Conn) {
	self.Connections[clientId] = NewConnection(conn)
}

func (self *Room) getConnection(clientId string) *Connection {
	return self.Connections[clientId]
}

type Connection struct {
	Conn  net.Conn
	State *ConnectionState
}

type ConnectionState struct {
	WaitingIds      []string
	WaitingMessages map[string]string
	PendingMessages map[string]string
}

type ChannelMessage struct {
	ClientId string
	Conn     net.Conn
	Message  codec.Message
}

func NewConnection(conn net.Conn) *Connection {
	return &Connection{conn, &ConnectionState{}}
}

var district *District

func GetDistrict() *District {
	if district == nil {
		district = NewDistrict(16)
	}
	return district
}
