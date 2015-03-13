package codec

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

type MessageBody interface {
	Encode(buffer *bytes.Buffer)
	Decode(r io.Reader) error
}

type Message struct {
	Hdr  *Header
	Body MessageBody
}

func (self *Header) Encode(buffer *bytes.Buffer) {
	b := byte(self.Type)<<4 | IfByte(self.Dup, byte(1), byte(0))<<3 | byte(self.Qos)<<1
	buffer.WriteByte(b)
	writeLength(buffer, self.TLength)
}

func (self *Header) Decode(r io.Reader) (err error) {
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
	if l, err = readLength(r, self); err != nil {
		return err
	}
	self.Type = messageType
	self.Dup = dup
	self.Qos = qos
	self.TLength = l
	self.RLength = l
	return nil
}

type Connect struct {
	Hdr             *Header
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

func (self *Connect) Encode(buffer *bytes.Buffer) {
	writeString(buffer, self.ProtocolName, self.Hdr)
	writeUint8(buffer, self.ProtocolVersion, self.Hdr)
	usernameFlag := IfInt(self.UsernameFlag, 1<<7, 0)
	passwordFlag := IfInt(self.PasswordFlag, 1<<6, 0)
	willRetain := IfInt(self.WillRetain, 1<<5, 0)
	willQos := IfInt(self.WillQos, 1<<3, 0)
	willFlag := IfInt(self.WillFlag, 1<<2, 0)
	cleanSession := IfInt(self.CleanSession, 1<<1, 0)
	flags := usernameFlag | passwordFlag | willRetain | willQos | willFlag | cleanSession
	writeUint8(buffer, uint8(flags), self.Hdr)
	writeUint16(buffer, self.KeepAliveTimer, self.Hdr)
	writeString(buffer, self.ClientId, self.Hdr)
	if self.WillFlag {
		writeString(buffer, self.WillTopic, self.Hdr)
		writeString(buffer, self.WillMessage, self.Hdr)
	}
	if self.UsernameFlag {
		writeString(buffer, self.Username, self.Hdr)
	}
	if self.PasswordFlag {
		writeString(buffer, self.Password, self.Hdr)
	}
}

func (self *Connect) Decode(r io.Reader) (err error) {
	var protocolName string
	var protocolVersion uint8
	var flags uint8
	var clientId string
	var keepAliveTimer uint16
	if protocolName, err = readString(r, self.Hdr); err != nil {
		return err
	}
	if protocolVersion, err = readUint8(r, self.Hdr); err != nil {
		return err
	}
	if flags, err = readUint8(r, self.Hdr); err != nil {
		return err
	}
	usernameFlag := b2bool(flags & 0x80)
	passwordFlag := b2bool(flags & 0x40)
	willRetain := b2bool(flags & 0x20)
	willQos := b2bool(flags & 0x18)
	willFlag := b2bool(flags & 0x04)
	cleanSession := b2bool(flags & 0x02)
	if keepAliveTimer, err = readUint16(r, self.Hdr); err != nil {
		return err
	}
	if clientId, err = readString(r, self.Hdr); err != nil {
		return err
	}
	if len(clientId) == 0 || len(clientId) >= 24 {
		return clientidError
	}
	var willTopic, willMessage, username, password string
	if willFlag {
		if willTopic, err = readString(r, self.Hdr); err != nil {
			return err
		}
		if willMessage, err = readString(r, self.Hdr); err != nil {
			return err
		}
	}
	if usernameFlag {
		if username, err = readString(r, self.Hdr); err != nil {
			return err
		}
	}
	if passwordFlag {
		if password, err = readString(r, self.Hdr); err != nil {
			return err
		}
	}
	self.ProtocolName = protocolName
	self.ProtocolVersion = protocolVersion
	self.UsernameFlag = usernameFlag
	self.PasswordFlag = passwordFlag
	self.WillRetain = willRetain
	self.WillQos = willQos
	self.WillFlag = willFlag
	self.CleanSession = cleanSession
	self.KeepAliveTimer = keepAliveTimer
	self.ClientId = clientId
	self.WillTopic = willTopic
	self.WillMessage = willMessage
	self.Username = username
	self.Password = password
	return
}

type ConnAck struct {
	Hdr        *Header
	ReturnCode ReturnCode
}

func (self *ConnAck) Encode(buffer *bytes.Buffer) {
	writeUint8(buffer, uint8(0), self.Hdr)
	writeUint8(buffer, uint8(self.ReturnCode), self.Hdr)
}

func (self *ConnAck) Decode(r io.Reader) (err error) {
	// pass reserved byte
	if _, err := readUint8(r, self.Hdr); err != nil {
		return err
	}
	var i uint8
	if i, err = readUint8(r, self.Hdr); err != nil {
		return err
	}
	rc := ReturnCode(i)
	if !rc.IsValid() {
		return rcError
	}
	self.ReturnCode = rc
	return nil
}

type Publish struct {
	Hdr       *Header
	MessageId uint16
	TopicName string
	Content   string
}

func (self *Publish) Encode(buffer *bytes.Buffer) {
	writeString(buffer, self.TopicName, self.Hdr)
	writeUint16(buffer, self.MessageId, self.Hdr)
	writeString(buffer, self.Content, self.Hdr)
}

func (self *Publish) Decode(r io.Reader) (err error) {
	var topicName, content string
	var messageId uint16
	if topicName, err = readString(r, self.Hdr); err != nil {
		return err
	}
	if self.Hdr.Qos.HasId() {
		if messageId, err = readUint16(r, self.Hdr); err != nil {
			return err
		}
	}
	if content, err = readString(r, self.Hdr); err != nil {
		return err
	}
	self.TopicName = topicName
	self.MessageId = messageId
	self.Content = content
	return nil
}

type PubAck struct {
	Hdr       *Header
	MessageId uint16
}

func (self *PubAck) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, self.MessageId, self.Hdr)
}

func (self *PubAck) Decode(r io.Reader) (err error) {
	if self.Hdr.Qos.HasId() {
		if self.MessageId, err = readUint16(r, self.Hdr); err != nil {
			return err
		}
	}
	return nil
}

type PubRec struct {
	Hdr       *Header
	MessageId uint16
}

func (self *PubRec) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, self.MessageId, self.Hdr)
}

func (self *PubRec) Decode(r io.Reader) (err error) {
	if self.Hdr.Qos.HasId() {
		if self.MessageId, err = readUint16(r, self.Hdr); err != nil {
			return err
		}
	}
	return nil
}

type PubRel struct {
	Hdr       *Header
	MessageId uint16
}

func (self *PubRel) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, self.MessageId, self.Hdr)
}

func (self *PubRel) Decode(r io.Reader) (err error) {
	if self.Hdr.Qos.HasId() {
		if self.MessageId, err = readUint16(r, self.Hdr); err != nil {
			return err
		}
	}
	return nil
}

type PubComp struct {
	Hdr       *Header
	MessageId uint16
}

func (self *PubComp) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, self.MessageId, self.Hdr)
}

func (self *PubComp) Decode(r io.Reader) (err error) {
	if self.Hdr.Qos.HasId() {
		if self.MessageId, err = readUint16(r, self.Hdr); err != nil {
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
	Hdr       *Header
	MessageId uint16
	Topics    []TopicQos
}

func (self *Subscribe) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, self.MessageId, self.Hdr)
	for _, topic := range self.Topics {
		writeString(buffer, topic.Name, self.Hdr)
		writeUint8(buffer, uint8(topic.Qos), self.Hdr)
	}
}

func (self *Subscribe) Decode(r io.Reader) (err error) {
	if self.Hdr.Qos.HasId() {
		if self.MessageId, err = readUint16(r, self.Hdr); err != nil {
			return err
		}
	}
	topics := make([]TopicQos, 0)
	for {
		if self.Hdr.RLength == 0 {
			break
		}
		var topicName string
		var qos uint8
		if topicName, err = readString(r, self.Hdr); err != nil {
			return err
		}
		if qos, err = readUint8(r, self.Hdr); err != nil {
			return err
		}
		topics = append(topics, TopicQos{topicName, QosLevel(qos)})
	}
	if len(topics) == 0 {
		return notopicError
	}
	self.Topics = topics
	return nil
}

type SubAck struct {
	Hdr       *Header
	MessageId uint16
	TopicsQos []QosLevel
}

func (self *SubAck) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, self.MessageId, self.Hdr)
	for _, qos := range self.TopicsQos {
		writeUint8(buffer, uint8(qos), self.Hdr)
	}
}

func (self *SubAck) Decode(r io.Reader) (err error) {
	if self.Hdr.Qos.HasId() {
		if self.MessageId, err = readUint16(r, self.Hdr); err != nil {
			return err
		}
	}
	topicsQos := make([]QosLevel, 0)
	var qos uint8
	for {
		if self.Hdr.RLength == 0 {
			break
		}
		if qos, err = readUint8(r, self.Hdr); err != nil {
			return err
		}
		topicsQos = append(topicsQos, QosLevel(qos))
	}
	if len(topicsQos) == 0 {
		return notopicError
	}
	self.TopicsQos = topicsQos
	return nil
}

type Unsubscribe struct {
	Hdr        *Header
	MessageId  uint16
	TopicsName []string
}

func (self *Unsubscribe) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, self.MessageId, self.Hdr)
	for _, topicName := range self.TopicsName {
		writeString(buffer, topicName, self.Hdr)
	}
}

func (self *Unsubscribe) Decode(r io.Reader) (err error) {
	if self.Hdr.Qos.HasId() {
		if self.MessageId, err = readUint16(r, self.Hdr); err != nil {
			return err
		}
	}
	topicsName := make([]string, 0)
	for {
		if self.Hdr.RLength == 0 {
			break
		}
		var topicName string
		if topicName, err = readString(r, self.Hdr); err != nil {
			return err
		}
		topicsName = append(topicsName, topicName)
	}
	if len(topicsName) == 0 {
		return notopicError
	}
	self.TopicsName = topicsName
	return nil
}

type UnsubAck struct {
	Hdr       *Header
	MessageId uint16
}

func (self *UnsubAck) Encode(buffer *bytes.Buffer) {
	writeUint16(buffer, self.MessageId, self.Hdr)
}

func (self *UnsubAck) Decode(r io.Reader) (err error) {
	if self.Hdr.Qos.HasId() {
		if self.MessageId, err = readUint16(r, self.Hdr); err != nil {
			return err
		}
	}
	return nil
}

type PingReq struct {
	Hdr *Header
}

func (self *PingReq) Encode(buffer *bytes.Buffer) {

}

func (self *PingReq) Decode(r io.Reader) error {
	return nil
}

type PingResp struct {
	Hdr *Header
}

func (self *PingResp) Encode(buffer *bytes.Buffer) {

}

func (self *PingResp) Decode(r io.Reader) error {
	return nil
}

type Disconnect struct {
	Hdr *Header
}

func (self *Disconnect) Encode(buffer *bytes.Buffer) {

}

func (self *Disconnect) Decode(r io.Reader) error {
	return nil
}

func NewMessageBody(hdr *Header) MessageBody {
	switch hdr.Type {
	case TypeConnect:
		mb := new(Connect)
		mb.Hdr = hdr
		return mb
	case TypeConnAck:
		mb := new(ConnAck)
		mb.Hdr = hdr
		return mb
	case TypePublish:
		mb := new(Publish)
		mb.Hdr = hdr
		return mb
	case TypePubAck:
		mb := new(PubAck)
		mb.Hdr = hdr
		return mb
	case TypePubRec:
		mb := new(PubRec)
		mb.Hdr = hdr
		return mb
	case TypePubRel:
		mb := new(PubRel)
		mb.Hdr = hdr
		return mb
	case TypePubComp:
		mb := new(PubComp)
		mb.Hdr = hdr
		return mb
	case TypeSubscribe:
		mb := new(Subscribe)
		mb.Hdr = hdr
		return mb
	case TypeSubAck:
		mb := new(SubAck)
		mb.Hdr = hdr
		return mb
	case TypeUnsubscribe:
		mb := new(Unsubscribe)
		mb.Hdr = hdr
		return mb
	case TypeUnsubAck:
		mb := new(UnsubAck)
		mb.Hdr = hdr
		return mb
	case TypePingReq:
		mb := new(PingReq)
		mb.Hdr = hdr
		return mb
	case TypePingResp:
		mb := new(PingResp)
		mb.Hdr = hdr
		return mb
	case TypeDisconnect:
		mb := new(Disconnect)
		mb.Hdr = hdr
		return mb
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

func (mqtt *MQTT) ReadLoop(handler func(message Message, err error) bool) {
	for {
		hdr := new(Header)
		if err := hdr.Decode(mqtt.r); err != nil {
			handler(Message{hdr, nil}, err)
			break
		}
		body := NewMessageBody(hdr)
		if err := body.Decode(mqtt.r); err != nil {
			handler(Message{hdr, nil}, err)
			break
		}
		if hdr.RLength != 0 {
			handler(Message{hdr, nil}, lenError)
			break
		}
		stopFlag := handler(Message{hdr, body}, nil)
		if stopFlag {
			break
		}
	}
}

func (self *MQTT) ReadIter() func() (Message, error) {
	return func() (Message, error) {
		hdr := new(Header)
		if err := hdr.Decode(self.r); err != nil {
			return Message{nil, nil}, err
		}
		body := NewMessageBody(hdr)
		if err := body.Decode(self.r); err != nil {
			return Message{hdr, nil}, err
		}
		if hdr.RLength != 0 {
			return Message{hdr, nil}, lenError
		}
		return Message{hdr, body}, nil
	}
}

func (self *MQTT) Write(message Message) error {
	hdr, body := message.Hdr, message.Body
	mbuffer := bytes.NewBuffer(make([]byte, 0))
	body.Encode(mbuffer)
	hdr.TLength = hdr.RLength
	hbuffer := bytes.NewBuffer(make([]byte, 0))
	hdr.Encode(hbuffer)
	if _, err := hbuffer.WriteTo(self.w); err != nil {
		return err
	}
	if _, err := mbuffer.WriteTo(self.w); err != nil {
		return err
	}
	return nil
}
