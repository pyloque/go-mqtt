package mqtt

import (
	"bytes"
	"reflect"
	"testing"
)

func testNormal(
	t *testing.T, message Message, hdr *Header,
	testcb func(message Message, hdr *Header, rmessage Message, rhdr *Header)) {
	buffer := bytes.NewBuffer(make([]byte, 0))
	mqtt_w := NewMQTTWriter(buffer)
	mqtt_w.Write(message, hdr)
	mqtt_r := NewMQTTReader(buffer)
	mqtt_r.ReadLoop(func(rhdr *Header, rmessage Message, err error) bool {
		if err != nil {
			t.Error(err.Error())
			return true
		}
		if hdr.Type != rhdr.Type || hdr.Qos != rhdr.Qos || hdr.Dup != rhdr.Dup {
			t.Error("Header mismatch")
			return true
		}
		testcb(message, hdr, rmessage, rhdr)
		return true
	})
}

func TestPack(t *testing.T) {
	generalcb := func(message Message, hdr *Header, rmessage Message, rhdr *Header) {
		it := reflect.TypeOf(message).Elem()
		rt := reflect.TypeOf(rmessage).Elem()
		v := reflect.ValueOf(message).Elem()
		rv := reflect.ValueOf(rmessage).Elem()
		if it != rt {
			t.Error("type mismatch")
			return
		}
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).CanInterface() {
				continue
			}
			fi := v.Field(i).Interface()
			rfi := rv.Field(i).Interface()
			if !reflect.DeepEqual(fi, rfi) {
				t.Errorf("%s mismatch", it.Field(i).Name)
				return
			}
		}
	}
	{
		message := Connect{
			hdr:             &Header{Type: TypeConnect, Qos: Qos1},
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
		testNormal(t, &message, message.hdr, generalcb)
	}
	{
		message := ConnAck{&Header{Type: TypeConnAck, Qos: Qos1}, ProtocolError}
		testNormal(t, &message, message.hdr, generalcb)
	}
	{
		message := Publish{
			hdr:       &Header{Type: TypePublish, Qos: Qos1},
			MessageId: 12345,
			TopicName: "testtopic",
			Content:   "abcdefghijklmnopqrstuvwxyz"}
		testNormal(t, &message, message.hdr, generalcb)
	}
	{
		message := PubAck{
			hdr:       &Header{Type: TypePubAck, Qos: Qos1},
			MessageId: 12345}
		testNormal(t, &message, message.hdr, generalcb)
	}
	{
		message := PubRec{
			hdr:       &Header{Type: TypePubRec, Qos: Qos1},
			MessageId: 12345}
		testNormal(t, &message, message.hdr, generalcb)
	}
	{
		message := PubRel{
			hdr:       &Header{Type: TypePubRel, Qos: Qos1},
			MessageId: 12345}
		testNormal(t, &message, message.hdr, generalcb)
	}
	{
		message := PubComp{
			hdr:       &Header{Type: TypePubComp, Qos: Qos1},
			MessageId: 12345}
		testNormal(t, &message, message.hdr, generalcb)
	}
	{
		message := Subscribe{
			hdr:       &Header{Type: TypeSubscribe, Qos: Qos1},
			MessageId: 12345,
			Topics:    []TopicQos{{"topic1", Qos1}, {"topic2", Qos2}}}
		testNormal(t, &message, message.hdr, generalcb)
	}
	{
		message := SubAck{
			hdr:       &Header{Type: TypeSubAck, Qos: Qos1},
			MessageId: 12345,
			TopicsQos: []QosLevel{Qos1, Qos2}}
		testNormal(t, &message, message.hdr, generalcb)
	}
	{
		message := Unsubscribe{
			hdr:        &Header{Type: TypeUnsubscribe, Qos: Qos1},
			MessageId:  12345,
			TopicsName: []string{"topic1", "topic2"}}
		testNormal(t, &message, message.hdr, generalcb)
	}
	{
		message := UnsubAck{
			hdr:       &Header{Type: TypeUnsubAck, Qos: Qos1},
			MessageId: 12345}
		testNormal(t, &message, message.hdr, generalcb)
	}
	{
		message := PingReq{&Header{Type: TypePingReq, Qos: Qos1}}
		testNormal(t, &message, message.hdr, generalcb)
	}
	{
		message := PingResp{&Header{Type: TypePingResp, Qos: Qos1}}
		testNormal(t, &message, message.hdr, generalcb)
	}
	{
		message := Disconnect{&Header{Type: TypeDisconnect, Qos: Qos1}}
		testNormal(t, &message, message.hdr, generalcb)
	}
}
