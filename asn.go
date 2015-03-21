// Copyright 2014-2015 Apptimist, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/apptimistco/asn/debug"
)

const (
	AsnStr   = "asn"
	ConnTO   = 200 * time.Millisecond
	MaxSegSz = 4096
	MoreFlag = uint16(1 << 15)

	WithDeadline = false
)

const (
	opened uint8 = iota
	provisional
	established
	closed
)

var ErrTooLarge = errors.New("exceeds MaxSegSz")

type asn struct {
	debug.Debug
	name struct {
		local, remote string
	}
	// Version adapts to peer
	version Version
	// State may be { opened, provisional, established, closed }
	state uint8
	// Keys to Open/Seal
	box *Box
	rx  struct {
		ch    chan *PDU
		err   error
		black []byte
		red   []byte
		going bool
	}
	tx struct {
		ch    chan pdubox
		err   error
		black []byte
		red   []byte
		going bool
	}
	conn  net.Conn
	repos *Repos
	acker acker
	time  struct {
		in, out time.Time
	}
}

// Pair box and pdu to support reset of box after Ack of Login
type pdubox struct {
	pdu *PDU
	box *Box
}

func (asn *asn) Init() {
	asn.version = Latest
	asn.rx.ch = make(chan *PDU, 4)
	asn.tx.ch = make(chan pdubox, 4)
	asn.rx.going = false
	asn.tx.going = false
	asn.rx.black = make([]byte, 0, MaxSegSz)
	asn.tx.black = make([]byte, 0, MaxSegSz)
	asn.rx.red = make([]byte, 0, MaxSegSz)
	asn.tx.red = make([]byte, 0, 2+MaxSegSz)
	asn.acker.Init()
}

func (asn *asn) Conn() net.Conn      { return asn.conn }
func (asn *asn) IsOpened() bool      { return asn.state == opened }
func (asn *asn) IsProvisional() bool { return asn.state == provisional }
func (asn *asn) IsEstablished() bool { return asn.state == established }
func (asn *asn) IsClosed() bool {
	return asn.conn == nil || asn.state == closed
}

// gorx receives, decrypts and reassembles segmented PDUs on the asn.Rx.Q
// until error, or EOF; then closes asn.Rx.Q when done.
func (asn *asn) gorx() {
	pdu := NewPDUBuf()
	defer func() {
		r := recover()
		pdu.Free()
		if r != nil {
			asn.rx.err = r.(error)
			if asn.rx.err != io.EOF {
				asn.Failure(debug.Depth(4), asn.rx.err)
			}
		}
		close(asn.rx.ch)
		asn.rx.going = false
	}()
	for {
		l := uint16(0)
		if pdu.File != nil && pdu.PB != nil {
			panic(os.ErrInvalid)
		}
		_, err := (NBOReader{asn}).ReadNBO(&l)
		if err != nil {
			panic(err)
		}
		n := l & ^MoreFlag
		if n > MaxSegSz {
			panic(ErrTooLarge)
		}
		if n == 0 {
			panic(os.ErrInvalid)
		}
		asn.rx.red = asn.rx.red[:0]
		_, err = asn.Read(asn.rx.red[:n])
		if err != nil {
			panic(err)
		}
		asn.rx.black = asn.rx.black[:0]
		b, err := asn.box.Open(asn.rx.black[:], asn.rx.red[:n])
		if err != nil {
			panic(err)
		}
		_, err = pdu.Write(b)
		if err != nil {
			panic(err)
		}
		if (l & MoreFlag) == 0 {
			asn.rx.ch <- pdu
			pdu = NewPDUBuf()
		} else if pdu.PB != nil {
			pdu.File = asn.repos.tmp.New()
			pdu.FN = pdu.File.Name()
			pdu.File.Write(pdu.PB.Bytes())
			pdu.PB.Free()
			pdu.PB = nil
		}
	}
}

// gotx pulls PDU from a channel, segments, and encrypts before sending through
// asn.conn. This stops and closes the connection on error or closed channel.
func (asn *asn) gotx() {
	const maxBlack = MaxSegSz - BoxOverhead
	defer func() {
		r := recover()
		if asn.conn != nil {
			asn.state = closed
			asn.conn.Close()
		}
		if r != nil {
			asn.tx.err = r.(error)
			asn.Diag(debug.Depth(4), asn.tx.err)
		}
		asn.tx.going = false
	}()
	for {
		pb, open := <-asn.tx.ch
		if !open {
			asn.Diag("quit pdutx")
			runtime.Goexit()
		}
		err := pb.pdu.Open()
		if err != nil {
			panic(err)
		}
		for n := pb.pdu.Len(); n > 0; n = pb.pdu.Len() {
			if n > maxBlack {
				n = maxBlack
			}
			asn.tx.black = asn.tx.black[:n]
			if _, err = pb.pdu.Read(asn.tx.black); err != nil {
				panic(err)
			}
			asn.tx.red = asn.tx.red[:2]
			asn.tx.red, err = pb.box.Seal(asn.tx.red, asn.tx.black)
			if err != nil {
				panic(err)
			}
			l := uint16(len(asn.tx.red[2:]))
			if pb.pdu.Len() > 0 {
				l |= MoreFlag
			}
			binary.BigEndian.PutUint16(asn.tx.red[:2], l)
			if _, err = asn.Write(asn.tx.red); err != nil {
				panic(err)
			}
		}
		pb.pdu.Free()
		pb.pdu = nil
		pb.box = nil
	}
}

func IsNetTimeout(err error) bool {
	e, ok := err.(net.Error)
	return ok && e.Timeout()
}

// Read full buffer from asn.conn unless preempted with state == closed.
func (asn *asn) Read(b []byte) (n int, err error) {
	for i := 0; n < len(b); n += i {
		if asn.IsClosed() {
			err = io.EOF
			asn.Diag("closed")
			return
		}
		if WithDeadline {
			asn.conn.SetReadDeadline(time.Now().Add(ConnTO))
		}
		i, err = asn.conn.Read(b[n:])
		if err != nil && !IsNetTimeout(err) {
			if asn.IsClosed() {
				err = io.EOF
			} else {
				asn.Diag(err)
			}
			return
		}
	}
	return
}

func (asn *asn) Reset() {
	asn.Diag(debug.Depth(2), "asn reset")
	if asn.conn != nil {
		if asn.state != closed {
			asn.state = closed
			asn.conn.Close()
		}
		asn.conn = nil
	}
	asn.box = nil
	asn.repos = nil
	asn.rx.black = asn.rx.black[:0]
	asn.tx.black = asn.tx.black[:0]
	asn.rx.red = asn.rx.red[:0]
	asn.tx.red = asn.tx.red[:0]
	asn.name.local = ""
	asn.name.remote = ""
	asn.Debug.Reset()
	asn.acker.Reset()
}

func (asn *asn) Set(v interface{}) error {
	switch t := v.(type) {
	case *Box:
		asn.box = t
	case net.Conn:
		asn.conn = t
		asn.state = opened
		go asn.gorx()
		go asn.gotx()
	case string:
		asn.name.remote = t
		asn.Debug.Set(fmt.Sprintf("%s(%s)", asn.name.local, t))
	case *Repos:
		asn.repos = t
	case Version:
		if asn.version > t {
			asn.version = t
		}
	default:
		return os.ErrInvalid
	}
	return nil
}

// Queue PDU for segmentation, encryption and transmission
func (asn *asn) Tx(pdu *PDU) {
	if asn == nil {
		asn.Diag(debug.Depth(2), "tried to Tx on freed asn")
		return
	}
	if asn.IsClosed() {
		asn.Diag(debug.Depth(2), "tried to Tx on closed asn")
		return
	}
	asn.tx.ch <- pdubox{pdu: pdu, box: asn.box}
}

// Version steps down to the peer.
func (asn *asn) Version() Version { return asn.version }

// Write full buffer unless preempted by Closed state.
func (asn *asn) Write(b []byte) (n int, err error) {
	for i := 0; n < len(b); n += i {
		if asn.IsClosed() {
			err = io.EOF
			asn.Diag("closed")
			return
		}
		if WithDeadline {
			asn.conn.SetWriteDeadline(time.Now().Add(ConnTO))
		}
		i, err = asn.conn.Write(b[n:])
		if err != nil && !IsNetTimeout(err) {
			asn.Diag(err)
			return
		}
	}
	return
}
