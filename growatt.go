package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"
)

type growattPacketType uint16

const (
	unknown growattPacketType = iota
	registers
	registersAck
	ping
	pingAck
	announce
	weirdBigPacket
	dongleIdentifyResponse
)

func typeOf(m modbusTCP) (growattPacketType, error) {
	if m.ProtocolIdentifier != ModbusProtocolGrowattV5 && m.ProtocolIdentifier != ModbusProtocolGrowattV6 {
		return unknown, errors.New("Unknown protocol:" + m.ProtocolIdentifier.String())
	}

	switch growattType(m.GrowattMessageID) {
	case dataID:
		if m.Length == 257 {
			return registers, nil
		}
		return unknown, nil

	case pingID:
		if m.Length == 12 || m.Length == 32 {
			return ping, nil
		}
		return unknown, nil
	case data3ID:
		if m.Length == 257 {
			return announce, nil
		}
		return unknown, nil

	case weird:
		if m.Length == 257 {
			return weirdBigPacket, nil
		}
		return unknown, nil

	case identifyID:
		return dongleIdentifyResponse, nil
	}
	return unknown, nil
}

//Xor will loop around the b key if is is shorter than the a key
func xor(a []byte, b []byte) []byte {
	res := make([]byte, len(a))
	for i := range a {
		res[i] = a[i] ^ b[i%len(b)]
	}
	return res
}

//These types appear to be valid for multiple versions of the protocol
type growattType uint8

const (
	identifyID growattType = 0x19
	pingID     growattType = 0x16
	dataID     growattType = 0x04
	data3ID    growattType = 0x03
	weird      growattType = 0x50
)

//TaggedRegister is a simple helper pair
type TaggedRegister struct {
	time.Time
	Registers
}

//Registers might not be complete or correct. The fields we are currently using are correct though
type Registers struct {
	Status    uint16
	Ppv       uint32
	Vpv1      uint16
	Ipv1      uint16
	Ppv1      uint32
	Vpv2      uint16
	Ipv2      uint16
	Ppv2      uint32
	Pac       uint32
	Vac       uint16
	Vac1      uint16
	Iac1      uint16
	Pac1      uint32
	Vac2      uint16
	Iac2      uint16
	Pac2      uint32
	Vac3      uint16
	Iac3      uint16
	Pac3      uint32
	EToday    uint32
	ETotal    uint32
	Tall      uint32
	Tmp       uint16
	ISOF      uint16
	GFCIF     uint16
	DCIF      uint16
	Vpvfault  uint16
	Vacfault  uint16
	Facfault  uint16
	Tmpfault  uint16
	Faultcode uint16
	IPMtemp   uint16
	Pbusvolt  uint16
	Nbusvolt  uint16
	Padding   [12]byte
	Epv1today uint32
	Epv1total uint32
	Epv2today uint32
	Epv2total uint32
	Epvtotal  uint32
	Rac       uint32
	ERactoday uint32
	Eractotal uint32
}

//TODO give this a proper name
func readRegStruct(s []byte) Registers {
	r := bytes.NewReader(s[71:])

	var g Registers
	//TODO in other cases we have littleendian, is this correct at all?
	binary.Read(r, binary.BigEndian, &g)

	return g
}
