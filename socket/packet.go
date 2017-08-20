// Socket package provides a concise, powerful and high-performance TCP socket.
//
// Copyright 2017 HenryLee. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package socket

import (
	"encoding/json"
	"sync"

	"github.com/henrylee2cn/goutil"

	"github.com/henrylee2cn/teleport/codec"
)

var packetStack = new(struct {
	freePacket *Packet
	mu         sync.Mutex
})

// GetPacket gets a *Packet form packet stack.
// Note:
//  bodyGetting is only for reading form connection;
//  settings are only for writing to connection.
func GetPacket(bodyGetting func(*Header) interface{}, settings ...PacketSetting) *Packet {
	packetStack.mu.Lock()
	p := packetStack.freePacket
	if p == nil {
		p = NewPacket(bodyGetting)
	} else {
		packetStack.freePacket = p.next
		p.Reset(bodyGetting, settings...)
	}
	packetStack.mu.Unlock()
	return p
}

// PutPacket puts a *Packet to packet stack.
func PutPacket(p *Packet) {
	packetStack.mu.Lock()
	p.Body = nil
	p.next = packetStack.freePacket
	packetStack.freePacket = p
	packetStack.mu.Unlock()
}

// Packet provides header and body's containers for receiving and sending packet.
type Packet struct {
	// HeaderCodec header codec id
	HeaderCodec byte
	// BodyCodec body codec id
	BodyCodec byte
	// header content
	Header *Header `json:"header"`
	// body content
	Body interface{} `json:"body"`
	// header length
	HeaderLength int64 `json:"header_length"`
	// body length
	BodyLength int64 `json:"body_length"`
	// HeaderLength + BodyLength
	Length int64 `json:"length"`
	// get body by header
	// Note:
	//  only for writing packet;
	//  should be nil when reading packet.
	bodyGetting func(*Header) interface{} `json:"-"`
	next        *Packet                   `json:"-"`
}

// NewPacket creates a new *Packet.
// Note:
//  bodyGetting is only for reading form connection;
//  settings are only for writing to connection.
func NewPacket(bodyGetting func(*Header) interface{}, settings ...PacketSetting) *Packet {
	var p = &Packet{
		Header:      new(Header),
		bodyGetting: bodyGetting,
		HeaderCodec: codec.NilCodecId,
		BodyCodec:   codec.NilCodecId,
	}
	for _, f := range settings {
		f(p)
	}
	return p
}

// Reset resets itself.
// Note:
//  bodyGetting is only for reading form connection;
//  settings are only for writing to connection.
func (p *Packet) Reset(bodyGetting func(*Header) interface{}, settings ...PacketSetting) {
	p.next = nil
	p.bodyGetting = bodyGetting
	p.Header.Reset()
	p.Body = nil
	p.HeaderLength = 0
	p.BodyLength = 0
	p.Length = 0
	p.HeaderCodec = codec.NilCodecId
	p.BodyCodec = codec.NilCodecId
	for _, f := range settings {
		f(p)
	}
}

// ResetBodyGetting resets the function of geting body.
func (p *Packet) ResetBodyGetting(bodyGetting func(*Header) interface{}) {
	p.bodyGetting = bodyGetting
}

func (p *Packet) loadBody() interface{} {
	if p.bodyGetting != nil {
		p.Body = p.bodyGetting(p.Header)
	} else {
		p.Body = nil
	}
	return p.Body
}

// String returns printing text.
func (p *Packet) String() string {
	b, _ := json.MarshalIndent(p, "", "  ")
	return goutil.BytesToString(b)
}

// HeaderCodecName returns packet header codec name.
func (p *Packet) HeaderCodecName() string {
	c, err := codec.GetById(p.HeaderCodec)
	if err != nil {
		return ""
	}
	return c.Name()
}

// BodyCodecName returns packet body codec name.
func (p *Packet) BodyCodecName() string {
	c, err := codec.GetById(p.BodyCodec)
	if err != nil {
		return ""
	}
	return c.Name()
}

// SetHeaderCodecByName sets header codec by name.
func (p *Packet) SetHeaderCodecByName(codecName string) {
	p.HeaderCodec = getCodecByName(codecName).Id()
}

// SetBodyCodecByName sets body codec by name.
func (p *Packet) SetBodyCodecByName(codecName string) {
	p.BodyCodec = getCodecByName(codecName).Id()
}

// PacketSetting sets Header field.
type PacketSetting func(*Packet)

// WithHeaderCodec sets header codec.
func WithHeaderCodec(codecName string) PacketSetting {
	c, _ := codec.GetByName(codecName)
	codecId := c.Id()
	return func(p *Packet) {
		p.HeaderCodec = codecId
	}
}

// WithBodyCodec sets body codec.
func WithBodyCodec(codecName string) PacketSetting {
	codecId := getCodecByName(codecName).Id()
	return func(p *Packet) {
		p.BodyCodec = codecId
	}
}

// WithBodyCodec sets body gzip level.
func WithBodyGzip(gzipLevel int32) PacketSetting {
	return func(p *Packet) {
		p.Header.Gzip = gzipLevel
	}
}

func getCodecByName(codecName string) codec.Codec {
	c, err := codec.GetByName(codecName)
	if err != nil {
		panic(err)
	}
	return c
}