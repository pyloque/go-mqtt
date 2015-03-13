package mqtt

import (
	"bytes"
	"reflect"
	"testing"
)

func testNormal(
	t *testing.T, message Message,
	testcb func(body MessageBody, rbody MessageBody)) {
	buffer := bytes.NewBuffer(make([]byte, 0))
	mqtt_w := NewMQTTWriter(buffer)
	mqtt_w.Write(message)
	mqtt_r := NewMQTTReader(buffer)
	hdr, body := message.hdr, message.body
	mqtt_r.ReadLoop(func(rmessage Message, err error) bool {
		if err != nil {
			t.Error(err.Error())
			return true
		}
		rhdr, rbody := rmessage.hdr, rmessage.body
		if hdr.Type != rhdr.Type || hdr.Qos != rhdr.Qos || hdr.Dup != rhdr.Dup {
			t.Error("Header mismatch")
			return true
		}
		testcb(body, rbody)
		return true
	})
}

func TestPack(t *testing.T) {
	generalcb := func(body MessageBody, rbody MessageBody) {
		it := reflect.TypeOf(body).Elem()
		rt := reflect.TypeOf(body).Elem()
		v := reflect.ValueOf(body).Elem()
		rv := reflect.ValueOf(body).Elem()
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
		body := Connect{
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
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
	{
		body := ConnAck{&Header{Type: TypeConnAck, Qos: Qos1}, ProtocolError}
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
	{
		body := Publish{
			hdr:       &Header{Type: TypePublish, Qos: Qos1},
			MessageId: 12345,
			TopicName: "testtopic",
			Content:   "abcdefghijklmnopqrstuvwxyz"}
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
	{
		body := PubAck{
			hdr:       &Header{Type: TypePubAck, Qos: Qos1},
			MessageId: 12345}
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
	{
		body := PubRec{
			hdr:       &Header{Type: TypePubRec, Qos: Qos1},
			MessageId: 12345}
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
	{
		body := PubRel{
			hdr:       &Header{Type: TypePubRel, Qos: Qos1},
			MessageId: 12345}
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
	{
		body := PubComp{
			hdr:       &Header{Type: TypePubComp, Qos: Qos1},
			MessageId: 12345}
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
	{
		body := Subscribe{
			hdr:       &Header{Type: TypeSubscribe, Qos: Qos1},
			MessageId: 12345,
			Topics:    []TopicQos{{"topic1", Qos1}, {"topic2", Qos2}}}
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
	{
		body := SubAck{
			hdr:       &Header{Type: TypeSubAck, Qos: Qos1},
			MessageId: 12345,
			TopicsQos: []QosLevel{Qos1, Qos2}}
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
	{
		body := Unsubscribe{
			hdr:        &Header{Type: TypeUnsubscribe, Qos: Qos1},
			MessageId:  12345,
			TopicsName: []string{"topic1", "topic2"}}
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
	{
		body := UnsubAck{
			hdr:       &Header{Type: TypeUnsubAck, Qos: Qos1},
			MessageId: 12345}
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
	{
		body := PingReq{&Header{Type: TypePingReq, Qos: Qos1}}
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
	{
		body := PingResp{&Header{Type: TypePingResp, Qos: Qos1}}
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
	{
		body := Disconnect{&Header{Type: TypeDisconnect, Qos: Qos1}}
		testNormal(t, Message{body.hdr, &body}, generalcb)
	}
}
