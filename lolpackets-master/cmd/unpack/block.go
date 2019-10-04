// TODO (Cameron): Move this to somewhere reasonable.

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
)

// This wiki page was helpful in understanding the block format.
// https://github.com/loldevs/leaguespec/wiki/General-Binary-Format
//
// The only significant change is that Type seems to be 2 bytes rather than the
// 1 (and sometimes first two bytes of payload) as described in wiki.

type Block struct {
	Marker uint8

	Time     float32
	TimeLong bool // whether time is absolute sec (4 bytes) or relative millis (1 byte)

	Type        uint16
	TypePresent bool // whether the type bytes are present

	Blockparam     uint32
	BlockparamLong bool // whether the long format (4 bytes) or short (1 byte) is used

	ContentLength     uint32
	ContentLengthLong bool // whether the long format (4 bytes) or short (1 byte) is used

	Content []byte
}

func ReadBlock(r io.Reader) (*Block, error) {
	b := &Block{}
	b.Marker = readUint8(r)

	b.TimeLong = (b.Marker&0x80 == 0)
	b.TypePresent = (b.Marker&0x40 == 0)
	b.BlockparamLong = (b.Marker&0x20 == 0)
	b.ContentLengthLong = (b.Marker&0x10 == 0)

	if b.TimeLong {
		read(r, &b.Time)
	} else {
		b.Time = float32(readUint8(r))
	}

	if b.ContentLengthLong {
		read(r, &b.ContentLength)
	} else {
		b.ContentLength = uint32(readUint8(r))
	}

	if b.TypePresent {
		b.Type = readUint16(r)
	}

	if b.BlockparamLong {
		read(r, &b.Blockparam)
	} else {
		b.Blockparam = uint32(readUint8(r))
	}

	b.Content = make([]byte, b.ContentLength)
	_, err := r.Read(b.Content)

	return b, err
}

func (b *Block) PrintSummary() {
	fmt.Printf("marker: %#x\ttime: %04.3f\ttype: %#x\tlength: %v\n", b.Marker, b.Time, b.Type, b.ContentLength)
}

func (b *Block) PrintContents() {
	start := 0
	for start < len(b.Content) {
		end := start + 2
		if end > len(b.Content) {
			end = len(b.Content)
		}
		fmt.Printf("%x ", b.Content[start:end])
		start = end
	}
	fmt.Println("")
}
func read(r io.Reader, data interface{}) {
	err := binary.Read(r, binary.LittleEndian, data)
	if err != nil {
		log.Fatalln(err)
	}
}

func readUint8(r io.Reader) uint8 {
	var res uint8
	read(r, &res)
	return res
}

func readUint16(r io.Reader) uint16 {
	var res uint16
	read(r, &res)
	return res
}
