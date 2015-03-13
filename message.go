package mqtt

import (
	"bytes"
	"errors"
	"io"
)

func IfInt(b bool, t, f int) int {
	if b {
		return t
	}
	return f
}

func IfByte(b bool, t, f byte) byte {
	if b {
		return t
	}
	return f
}

func b2bool(b byte) bool {
	return b != 0
}

type QosLevel uint8

const (
	Qos0 = QosLevel(iota)
	Qos1
	Qos2
)

func (qos QosLevel) IsValid() bool {
	return qos <= Qos2
}

func (qos QosLevel) HasId() bool {
	return qos == Qos1 || qos == Qos2
}

type MessageType uint8

const (
	TypeReserved = MessageType(iota)
	TypeConnect
	TypeConnAck
	TypePublish
	TypePubAck
	TypePubRec
	TypePubRel
	TypePubComp
	TypeSubscribe
	TypeSubAck
	TypeUnsubscribe
	TypeUnsubAck
	TypePingReq
	TypePingResp
	TypeDisconnect
)

func (mt MessageType) IsValid() bool {
	return mt > TypeReserved && mt <= TypeDisconnect
}

type ReturnCode uint8

const (
	Ok = ReturnCode(iota)
	ProtocolError
	IdentifierError
	ServerUnavalable
	AuthError
	NoAuth
)

func (rc ReturnCode) IsValid() bool {
	return rc <= NoAuth
}

var (
	eofError      = errors.New("mqtt: eof")
	lenError      = errors.New("mqtt: length mismatch")
	typeError     = errors.New("mqtt: illegal message type")
	rcError       = errors.New("mqtt: illegal return code")
	qosError      = errors.New("mqtt: illegal qos level")
	notopicError  = errors.New("mqtt: topics not provided")
	clientidError = errors.New("mqtt: illegal clientId")
	writeError    = errors.New("mqtt: write error")
)

type Header struct {
	Type    MessageType
	Qos     QosLevel
	Dup     bool
	TLength uint32 // RemainingLength
	RLength uint32 // length of bytes unread in this current message, initialize with TLength
}

type Message interface {
	Encode(buffer *bytes.Buffer)
	Decode(r io.Reader) error
}

func (hdr *Header) Encode(buffer *bytes.Buffer) {
	b := byte(hdr.Type)<<4 | IfByte(hdr.Dup, byte(1), byte(0))<<3 | byte(hdr.Qos)<<1
	buffer.WriteByte(b)
	writeLength(buffer, hdr.TLength)
}

func (hdr *Header) Decode(r io.Reader) (err error) {
	buf := make([]byte, 1)
	if n, err := r.Read(buf); n == 0 {
		return err
	}
	b := buf[0]
	messageType := MessageType((b & 0xF0) >> 4)
	if !messageType.IsValid() {
		return typeError
	}
	dup := b2bool(b & 0x08 >> 3)
	qos := QosLevel(b & 0x06 >> 1)
	if !qos.IsValid() {
		return qosError
	}
	var l uint32
	if l, err = readLength(r, hdr); err != nil {
		return err
	}
	hdr.Type = messageType
	hdr.Dup = dup
	hdr.Qos = qos
	hdr.TLength = l
	hdr.RLength = l
	return nil
}

type Connect struct {
	hdr             *Header
	ProtocolName    string
	ProtocolVersion uint8
	UsernameFlag    bool
	PasswordFlag    bool
	WillRetain      bool
	WillQos         bool
	WillFlag        bool
	CleanSession    bool
	KeepAliveTimer  uint16
	ClientId        string
	WillTopic       string
	WillMessage     string
	Username        string
	Password        string
}

func (message *Connect) Encode(buffer *bytes.Buffer) {
	writeString(buffer, message.ProtocolName, message.hdr)
	writeUint8(buffer, message.ProtocolVersion, message.hdr)
	usernameFlag := IfInt(message.UsernameFlag, 1<<7, 0)
	passwordFlag := IfInt(message.PasswordFlag, 1<<6, 0)
	willRetain := IfInt(message.WillRetain, 1<<5, 0)
	willQos := IfInt(message.WillQos, 1<<3, 0)
	willFlag := IfInt(message.WillFlag, 1<<2, 0)
	cleanSession := IfInt(message.CleanSession, 1<<1, 0)
	flags := usernameFlag | passwordFlag | willRetain | willQos | willFlag | cleanSession
	writeUint8(buffer, uint8(flags), message.hdr)
	writeUint16(buffer, message.KeepAliveTimer, message.hdr)
	writeString(buffer, message.ClientId, message.hdr)
	if message.WillFlag {
		writeString(buffer, message.WillTopic, message.hdr)
		writeString(buffer, message.WillMessage, message.hdr)
	}
	if message.UsernameFlag {
		writeString(buffer, message.Username, message.hdr)
	}
	if message.PasswordFlag {
		writeString(buffer, message.Password, message.hdr)
	}
}

func (message *Connect) Decode(r io.Reader) (err error) {
	var protocolName string
	var protocolVersion uint8
	var flags uint8
	var clientId string
	var keepAliveTimer uint16
	if protocolName, err = readString(r, message.hdr); err != nil {
		return err
	}
	if protocolVersion, err = readUint8(r, message.hdr); err != nil {
		return err
	}
	if flags, err = readUint8(r, message.hdr); err != nil {
		return err
	}
	usernameFlag := b2bool(flags & 0x80)
	passwordFlag := b2bool(flags & 0x40)
	willRetain := b2bool(flags & 0x20)
	willQos := b2bool(flags & 0x18)
	willFlag := b2bool(flags & 0x04)
	cleanSession := b2bool(flags & 0x02)
	if keepAliveTimer, err = readUint16(r, message.hdr); err != nil {
		return err
	}
	if clientId, err = readString(r, message.hdr); err != nil {
		return err
	}
	if len(clientId) == 0 || len(clientId) >= 24 {
		return clientidError
	}
	var willTopic, willMessage, username, password string
	if willFlag {
		if willTopic, err = readString(r, message.hdr); err != nil {
			return err
		}
		if willMessage, err = readString(r, message.hdr); err != nil {
			return err
		}
	}
	if usernameFlag {
		if username, err = readString(r, message.hdr); err != nil {
			return err
		}
	}
	if passwordFlag {
		if password, err = readString(r, message.hdr); err != nil {
			return err
		}
	}
	message.ProtocolName = protocolName
	message.ProtocolVersion = protocolVersion
	message.UsernameFlag = usernameFlag
	message.PasswordFlag = passwordFlag
	message.WillRetain = willRetain
	message.WillQos = willQos
	message.WillFlag = willFlag
	message.CleanSession = cleanSession
	message.KeepAliveTimer = keepAliveTimer
	message.ClientId = clientId
	message.WillTopic = willTopic
	message.WillMessage = willMessage
	message.Username = username
	message.Password = password
	return
}

type ConnAck struct {
	hdr        *Header
	ReturnCode ReturnCode
}

func (message *ConnAck) Encode(buffer *bytes.Buffer) {
	writeUint8(buffer, uint8(0), message.hdr)
	writeUint8(buffer, uint8(message.ReturnCode), message.hdr)
}

func (message *ConnAck) Decode(r io.Reader) (err error) {
	// pass reserved byte
	if _, err := readUint8(r, message.hdr); err != nil {
		return err
	}
	var i uint8
	if i, err = readUint8(r, message.hdr); err != nil {
		return err
	}
	rc := ReturnCode(i)
	if !rc.IsValid() {
		return rcError
	}
	message.ReturnCode = rc
	return nil
}

type Publish struct {
	hdr       *Header
	MessageId uint16
	TopicName string
	Content   string
}

func (message *Publish) Encode(buffer *bytes.Buffer) {
	writeString(buffer, message.TopicName, message.hdr)
	writeUint16(buffer, message.MessageId, message.hdr)
	writeString(buffer, message.Content, message.hdr)
}

func (message *Publish) Decode(r io.Reader) (err error) {
	var topicName, content string
	var messageId uint16
	if topicName, err = readString(r, message.hdr); err != nil {
		return err
	}
	if message.hdr.Qos.HasId() {
		if messageId, err = readUint16(r, message.hdr); err != nil {
			return err
		}
	}
	if content, err = readString(r, message.hdr); err != nil {
		return err
	}
	message.TopicName = topicName
	message.MessageId = messageId
	message.Content = content
	return nil
}

type PubAck struct {
	hdr       *Header
	MessageId uint16
}

func (message *PubAck) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, message.MessageId, message.hdr)
}

func (message *PubAck) Decode(r io.Reader) (err error) {
	if message.hdr.Qos.HasId() {
		if message.MessageId, err = readUint16(r, message.hdr); err != nil {
			return err
		}
	}
	return nil
}

type PubRec struct {
	hdr       *Header
	MessageId uint16
}

func (message *PubRec) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, message.MessageId, message.hdr)
}

func (message *PubRec) Decode(r io.Reader) (err error) {
	if message.hdr.Qos.HasId() {
		if message.MessageId, err = readUint16(r, message.hdr); err != nil {
			return err
		}
	}
	return nil
}

type PubRel struct {
	hdr       *Header
	MessageId uint16
}

func (message *PubRel) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, message.MessageId, message.hdr)
}

func (message *PubRel) Decode(r io.Reader) (err error) {
	if message.hdr.Qos.HasId() {
		if message.MessageId, err = readUint16(r, message.hdr); err != nil {
			return err
		}
	}
	return nil
}

type PubComp struct {
	hdr       *Header
	MessageId uint16
}

func (message *PubComp) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, message.MessageId, message.hdr)
}

func (message *PubComp) Decode(r io.Reader) (err error) {
	if message.hdr.Qos.HasId() {
		if message.MessageId, err = readUint16(r, message.hdr); err != nil {
			return err
		}
	}
	return nil
}

type TopicQos struct {
	Name string
	Qos  QosLevel
}

type Subscribe struct {
	hdr       *Header
	MessageId uint16
	Topics    []TopicQos
}

func (message *Subscribe) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, message.MessageId, message.hdr)
	for _, topic := range message.Topics {
		writeString(buffer, topic.Name, message.hdr)
		writeUint8(buffer, uint8(topic.Qos), message.hdr)
	}
}

func (message *Subscribe) Decode(r io.Reader) (err error) {
	if message.hdr.Qos.HasId() {
		if message.MessageId, err = readUint16(r, message.hdr); err != nil {
			return err
		}
	}
	topics := make([]TopicQos, 0)
	for {
		if message.hdr.RLength == 0 {
			break
		}
		var topicName string
		var qos uint8
		if topicName, err = readString(r, message.hdr); err != nil {
			return err
		}
		if qos, err = readUint8(r, message.hdr); err != nil {
			return err
		}
		topics = append(topics, TopicQos{topicName, QosLevel(qos)})
	}
	if len(topics) == 0 {
		return notopicError
	}
	message.Topics = topics
	return nil
}

type SubAck struct {
	hdr       *Header
	MessageId uint16
	TopicsQos []QosLevel
}

func (message *SubAck) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, message.MessageId, message.hdr)
	for _, qos := range message.TopicsQos {
		writeUint8(buffer, uint8(qos), message.hdr)
	}
}

func (message *SubAck) Decode(r io.Reader) (err error) {
	if message.hdr.Qos.HasId() {
		if message.MessageId, err = readUint16(r, message.hdr); err != nil {
			return err
		}
	}
	topicsQos := make([]QosLevel, 0)
	var qos uint8
	for {
		if message.hdr.RLength == 0 {
			break
		}
		if qos, err = readUint8(r, message.hdr); err != nil {
			return err
		}
		topicsQos = append(topicsQos, QosLevel(qos))
	}
	if len(topicsQos) == 0 {
		return notopicError
	}
	message.TopicsQos = topicsQos
	return nil
}

type Unsubscribe struct {
	hdr        *Header
	MessageId  uint16
	TopicsName []string
}

func (message *Unsubscribe) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, message.MessageId, message.hdr)
	for _, topicName := range message.TopicsName {
		writeString(buffer, topicName, message.hdr)
	}
}

func (message *Unsubscribe) Decode(r io.Reader) (err error) {
	if message.hdr.Qos.HasId() {
		if message.MessageId, err = readUint16(r, message.hdr); err != nil {
			return err
		}
	}
	topicsName := make([]string, 0)
	for {
		if message.hdr.RLength == 0 {
			break
		}
		var topicName string
		if topicName, err = readString(r, message.hdr); err != nil {
			return err
		}
		topicsName = append(topicsName, topicName)
	}
	if len(topicsName) == 0 {
		return notopicError
	}
	message.TopicsName = topicsName
	return nil
}

type UnsubAck struct {
	hdr       *Header
	MessageId uint16
}

func (message *UnsubAck) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, message.MessageId, message.hdr)
}

func (message *UnsubAck) Decode(r io.Reader) (err error) {
	if message.hdr.Qos.HasId() {
		if message.MessageId, err = readUint16(r, message.hdr); err != nil {
			return err
		}
	}
	return nil
}

type PingReq struct {
	hdr *Header
}

func (message *PingReq) Encode(buffer *bytes.Buffer) {

}

func (message *PingReq) Decode(r io.Reader) error {
	return nil
}

type PingResp struct {
	hdr *Header
}

func (message *PingResp) Encode(buffer *bytes.Buffer) {

}

func (message *PingResp) Decode(r io.Reader) error {
	return nil
}

type Disconnect struct {
	hdr *Header
}

func (message *Disconnect) Encode(buffer *bytes.Buffer) {

}

func (message *Disconnect) Decode(r io.Reader) error {
	return nil
}

func NewMessage(hdr *Header) Message {
	switch hdr.Type {
	case TypeConnect:
		message := new(Connect)
		message.hdr = hdr
		return message
	case TypeConnAck:
		message := new(ConnAck)
		message.hdr = hdr
		return message
	case TypePublish:
		message := new(Publish)
		message.hdr = hdr
		return message
	case TypePubAck:
		message := new(PubAck)
		message.hdr = hdr
		return message
	case TypePubRec:
		message := new(PubRec)
		message.hdr = hdr
		return message
	case TypePubRel:
		message := new(PubRel)
		message.hdr = hdr
		return message
	case TypePubComp:
		message := new(PubComp)
		message.hdr = hdr
		return message
	case TypeSubscribe:
		message := new(Subscribe)
		message.hdr = hdr
		return message
	case TypeSubAck:
		message := new(SubAck)
		message.hdr = hdr
		return message
	case TypeUnsubscribe:
		message := new(Unsubscribe)
		message.hdr = hdr
		return message
	case TypeUnsubAck:
		message := new(UnsubAck)
		message.hdr = hdr
		return message
	case TypePingReq:
		message := new(PingReq)
		message.hdr = hdr
		return message
	case TypePingResp:
		message := new(PingResp)
		message.hdr = hdr
		return message
	case TypeDisconnect:
		message := new(Disconnect)
		message.hdr = hdr
		return message
	}
	return nil
}

func readLength(r io.Reader, hdr *Header) (l uint32, err error) {
	buf := make([]byte, 1)
	var i int
	var p uint8
	for {
		var n int
		if n, err = r.Read(buf); n == 0 {
			return 0, err
		}
		num := uint8(buf[0])
		flag, digit := b2bool(num&0x80), num&0x7F
		l += uint32(digit) << p
		i++
		p += 7
		if !flag { // stop if continue bit == 0
			break
		}
		if i >= 4 { // max 4 bytes
			if flag {
				return 0, lenError
			}
			break
		}
	}
	return l, nil
}

func writeLength(buffer *bytes.Buffer, l uint32) {
	for {
		flag, digit := b2bool(byte(l&0x80)), l&0x7F
		buffer.WriteByte(byte(digit))
		if !flag {
			break
		}
		l = l >> 7
	}
}

func readString(r io.Reader, hdr *Header) (s string, err error) {
	var l uint16
	if l, err = readUint16(r, hdr); err != nil {
		return "", lenError
	}
	if hdr.RLength < uint32(l) {
		return "", lenError
	}
	buf := make([]byte, l)
	n, err := r.Read(buf)
	if n != len(buf) {
		return "", eofError
	}
	hdr.RLength -= uint32(l)
	return string(buf), nil
}

func writeString(buffer *bytes.Buffer, s string, hdr *Header) {
	writeUint16(buffer, uint16(len(s)), hdr)
	buffer.WriteString(s)
	hdr.RLength += uint32(len(s))
}

func readUint8(r io.Reader, hdr *Header) (uint8, error) {
	if hdr.RLength == 0 {
		return 0, lenError
	}
	buf := make([]byte, 1)
	if n, _ := r.Read(buf); n == 0 {
		return 0, eofError
	}
	hdr.RLength--
	return uint8(buf[0]), nil
}

func writeUint8(buffer *bytes.Buffer, i uint8, hdr *Header) {
	buffer.WriteByte(byte(i))
	hdr.RLength++
}

func readUint16(r io.Reader, hdr *Header) (uint16, error) {
	if hdr.RLength <= 1 {
		return 0, lenError
	}
	buf := make([]byte, 2)
	if n, _ := r.Read(buf); n != 2 {
		return 0, eofError
	}
	hdr.RLength -= 2
	d := uint16(buf[0])<<8 + uint16(buf[1])
	return d, nil
}

func writeUint16(buffer *bytes.Buffer, i uint16, hdr *Header) {
	buffer.WriteByte(byte(i >> 8))
	buffer.WriteByte(byte(i & 0xFF))
	hdr.RLength += 2
}

type MQTT struct {
	r io.Reader
	w io.Writer
}

func NewMQTTWriter(w io.Writer) *MQTT {
	return &MQTT{w: w}
}

func NewMQTTReader(r io.Reader) *MQTT {
	return &MQTT{r: r}
}

func (mqtt *MQTT) ReadLoop(handler func(hdr *Header, message Message, err error) bool) {
	/***
		mqtt.ReadLoop(func(hdr *Header, message Message, err error) {
			if err == nil {
				fmt.Println(hdr.Type)
			}
		})
	***/
	for {
		hdr := new(Header)
		if err := hdr.Decode(mqtt.r); err != nil {
			handler(hdr, nil, err)
			break
		}
		message := NewMessage(hdr)
		if err := message.Decode(mqtt.r); err != nil {
			handler(hdr, nil, err)
			break
		}
		if hdr.RLength != 0 {
			handler(hdr, nil, lenError)
			break
		}
		stopFlag := handler(hdr, message, nil)
		if stopFlag {
			break
		}
	}
}

func (mqtt *MQTT) ReadIter() func() (*Header, Message, error) {
	/***
		it := ReadIter(r)
		for {
			if hdr, message, err := it(); err != nil {
				break
			}
			fmt.Println(hdr.Type)
		}
	***/
	return func() (*Header, Message, error) {
		hdr := new(Header)
		if err := hdr.Decode(mqtt.r); err != nil {
			return nil, nil, err
		}
		message := NewMessage(hdr)
		if err := message.Decode(mqtt.r); err != nil {
			return hdr, nil, err
		}
		if hdr.RLength != 0 {
			return hdr, nil, lenError
		}
		return hdr, message, nil
	}
}

func (mqtt *MQTT) Write(message Message, hdr *Header) error {
	mbuffer := bytes.NewBuffer(make([]byte, 0))
	message.Encode(mbuffer)
	hdr.TLength = hdr.RLength
	hbuffer := bytes.NewBuffer(make([]byte, 0))
	hdr.Encode(hbuffer)
	if _, err := hbuffer.WriteTo(mqtt.w); err != nil {
		return err
	}
	if _, err := mbuffer.WriteTo(mqtt.w); err != nil {
		return err
	}
	return nil
}
