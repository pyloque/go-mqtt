package main

import (
	"log"
	"net"

	"github.com/pyloque/mqtt/codec"
)

func StartByIndex(protocol string, addr string, index int) {
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Client %d Started\n", index)
	go func(conn net.Conn) {
		mqtt := codec.NewMQTTWriter(conn)
		{
			body := codec.Connect{
				Hdr:             &codec.Header{Type: codec.TypeConnect, Qos: codec.Qos1},
				ProtocolName:    "MQIsdp",
				ProtocolVersion: 3,
				UsernameFlag:    true,
				PasswordFlag:    true,
				WillRetain:      true,
				WillQos:         true,
				WillFlag:        true,
				CleanSession:    true,
				KeepAliveTimer:  111,
				ClientId:        "abcd",
				WillTopic:       "testtopic",
				WillMessage:     "testmessage",
				Username:        "testuser",
				Password:        "testpassword"}
			message := codec.Message{body.Hdr, &body}
			mqtt.Write(message)
		}
		{
			body := codec.Publish{
				Hdr:       &codec.Header{Type: codec.TypePublish, Qos: codec.Qos1},
				MessageId: 12345,
				TopicName: "testtopic",
				Content:   "abcdefghijklmnopqrstuvwxyz"}
			message := codec.Message{body.Hdr, &body}
			mqtt.Write(message)
		}
		{
			body := codec.Disconnect{
				Hdr: &codec.Header{Type: codec.TypeConnect, Qos: codec.Qos1},
			}
			message := codec.Message{body.Hdr, &body}
			mqtt.Write(message)
		}
		conn.Close()
	}(conn)
}

func Start(protocol string, addr string) {
	for i := 0; i < 10; i++ {
		StartByIndex(protocol, addr, i)
	}
}

func main() {
	Start("tcp", ":2222")
}
