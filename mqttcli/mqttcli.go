package main

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/pyloque/mqtt/codec"
)

func StartByIndex(protocol string, addr string, index int, wg *sync.WaitGroup) {
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Client %d Started\n", index)
	wg.Add(1)
	go func(conn net.Conn, index int) {
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
				Hdr: &codec.Header{Type: codec.TypeDisconnect, Qos: codec.Qos1},
			}
			message := codec.Message{body.Hdr, &body}
			mqtt.Write(message)
		}
		time.Sleep(1000 * time.Second)
		conn.Close()
		wg.Done()
		log.Printf("Client %d Ends", index)
	}(conn, index)
}

func Start(protocol string, addr string) {
	wg := &sync.WaitGroup{}
	for i := 0; i < 100000; i++ {
		StartByIndex(protocol, addr, i, wg)
	}
	wg.Wait()
}

func main() {
	Start("tcp", ":2222")
}
