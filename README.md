Write Message
================

```
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
	buffer := bytes.NewBuffer(make([]byte, 0))
	mqtt_w := NewMQTTWriter(buffer)
	mqtt_w.Write(message, hdr)
```

Read Message
=================

```
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
		return true
	})
```
