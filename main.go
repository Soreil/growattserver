package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

var systemID int
var apiKey string

func init() {
	flag.IntVar(&systemID, "id", 0, "PVOutput system ID")
	flag.StringVar(&apiKey, "key", "", "PVOutput API key")
	flag.Parse()
	if systemID == 0 {
		log.Fatalln("System ID is a required argument")
	}
	if apiKey == "" {
		log.Fatalln("API key is a required argument")
	}
}

func handle(c net.Conn) {
	defer c.Close()

	tmp := make([]byte, 1024)
	data := make([]byte, 0)

	readHeader := false
	readBody := false

	for {
		// read to the tmp var
		n, err := c.Read(tmp)
		if err != nil {
			// log if not normal error
			if err != io.EOF {
				fmt.Printf("Read error - %s\n", err)
			}
			break
		}

		// append read data to full data
		data = append(data, tmp[:n]...)

		var modbus modbusTCP
		if !readHeader && len(data) >= 8 {
			if err := modbus.decodeHeaders(data); err != nil {
				log.Println(err)
				break
			}
			data = data[8:]
			readHeader = true
		}

		if !readBody && len(data) >= int(modbus.Length) {
			modbus.payload = data[:int(modbus.Length)]
			data = data[int(modbus.Length):]
			readBody = true
		}

		if readBody && readHeader {
			t, err := typeOf(modbus)
			if err != nil {
				log.Println(err)
				break
			}

			switch t {
			case weirdBigPacket:
				fmt.Println("that odd 0x50 config data packet thing")
				sendReplyWithCRC(modbus, []byte{0x47}, c)
			case registers:
				f := func(payload []byte) {
					const XORKey = "Growatt"
					body := xor(payload, []byte(XORKey))
					regs := readRegStruct(body)
					if err := upload(TaggedRegister{time.Now(), regs}); err != nil {
						log.Println(err)
					}
				}
				fmt.Println("solar panel data yatta")
				sendReplyWithCRC(modbus, []byte{0x47}, c)
				go f(modbus.payload)
			case ping:
				fmt.Println("ping")
				msg := append(makeHeader(modbus, modbus.Length), modbus.payload...)
				_, err := c.Write(msg)
				if err != nil {
					log.Println(err)
				}
			case announce:
				log.Println("Announce")
				sendReplyWithCRC(modbus, []byte{0x47}, c)
			default:
				log.Printf("Illegal packet type:%+v", modbus)
			}
			readHeader = false
			readBody = false
		}
	}
}

func sendReplyWithCRC(header modbusTCP, body []byte, c net.Conn) {
	//the +2 is for the CRC size
	msg := append(makeHeader(header, uint16(len(body)+2)), body...)
	high, low := computeCRC(msg)
	msg = append(msg, high)
	msg = append(msg, low)

	_, err := c.Write(msg)
	if err != nil {
		log.Println(err)
	}
	log.Printf("Wrote %d bytes\n", len(msg))
}

func makeHeader(m modbusTCP, length uint16) []byte {
	headerBuffer := make([]byte, 8)

	binary.BigEndian.PutUint16(headerBuffer[0:], m.TransactionIdentifier)
	binary.BigEndian.PutUint16(headerBuffer[2:], uint16(m.ProtocolIdentifier))
	binary.BigEndian.PutUint16(headerBuffer[4:], length)
	headerBuffer[6] = m.UnitIdentifier
	headerBuffer[7] = m.GrowattMessageID

	return headerBuffer
}

const port = 5279

func main() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handle(conn)
	}
}
