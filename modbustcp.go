package main

import (
	"encoding/binary"
	"errors"
)

const mbapRecordSizeInBytes int = 7
const growattRecordSizeInBytes int = mbapRecordSizeInBytes + 1
const modbusPDUMinimumRecordSizeInBytes int = 2
const modbusPDUMaximumRecordSizeInBytes int = 293
const growattPadding int = 2

// modbusProtocol type
type modbusProtocol uint16

// ModbusProtocol known values.
const (
	ModbusProtocolModbus    modbusProtocol = 0
	ModbusProtocolGrowattV5 modbusProtocol = 5
	ModbusProtocolGrowattV6 modbusProtocol = 6
)

var lut []uint16

func init() {
	lut = makeLUT()
}

func (mp modbusProtocol) String() string {
	switch mp {
	default:
		return "Unknown"
	case ModbusProtocolModbus:
		return "Modbus"
	case ModbusProtocolGrowattV5:
		return "GrowattV5"
	case ModbusProtocolGrowattV6:
		return "GrowattV6"
	}
}

// represents in a structured form the MODBUS Application Protocol header (MBAP) record present as the TCP
// payload in an modbusTCP TCP packet.
type modbusTCP struct {
	payload []byte

	TransactionIdentifier uint16         // Identification of a MODBUS Request/Response transaction
	ProtocolIdentifier    modbusProtocol // It is used for intra-system multiplexing
	Length                uint16         // Number of following bytes (includes 1 byte for UnitIdentifier + Modbus data length
	UnitIdentifier        uint8          // Identification of a remote slave connected on a serial line or on other buses
	GrowattMessageID      uint8
}

// DecodeFromBytes analyses a byte slice and attempts to decode it as an ModbusTCP
// record of a TCP packet.
func (d *modbusTCP) decodeHeaders(data []byte) error {

	// If the data block is too short to be a MBAP record, then return an error.
	if len(data) < mbapRecordSizeInBytes+modbusPDUMinimumRecordSizeInBytes {
		return errors.New("ModbusTCP packet too short")
	}

	if len(data) > mbapRecordSizeInBytes+modbusPDUMaximumRecordSizeInBytes {
		return errors.New("ModbusTCP packet too long")
	}

	// Extract the fields from the block of bytes.
	// The fields can just be copied in big endian order.
	d.TransactionIdentifier = binary.BigEndian.Uint16(data[:2])
	d.ProtocolIdentifier = modbusProtocol(binary.BigEndian.Uint16(data[2:4]))
	d.Length = binary.BigEndian.Uint16(data[4:6])

	d.UnitIdentifier = uint8(data[6])
	d.GrowattMessageID = uint8(data[7])

	return nil
}

//makeLUT is only called once since the value is read only and saved in a global on boot
func makeLUT() []uint16 {
	var LUT []uint16
	for i := uint16(0); i < 256; i++ {
		b := i
		var crc uint16
		for j := 0; j < 8; j++ {
			if ((b ^ crc) & 0x0001) == 1 {
				crc = (crc >> 1) ^ 0xa001
			} else {
				crc >>= 1
			}
			b >>= 1
		}
		LUT = append(LUT, crc)

	}
	return LUT
}

//computeCRC takes in the entire packet and returns a 2 byte value
func computeCRC(s []byte) (byte, byte) {
	crc := uint16(0xffff)
	for _, v := range s {
		idx := lut[(crc^uint16(v))&0xff]
		crc = ((crc >> 8) & 0xff) ^ idx
	}
	return byte((crc & 0xff00) >> 8), byte(crc & 0xff)
}
