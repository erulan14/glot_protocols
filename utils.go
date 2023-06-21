package main

import (
	"bytes"
	"encoding/binary"
)

type BitBuffer struct {
	Data   []byte
	Offset uint
}

func NewBitBuffer(data []byte) *BitBuffer {
	return &BitBuffer{
		Data:   data,
		Offset: 0,
	}
}

func (b *BitBuffer) ReadUnsigned(length uint) uint {
	value := uint(0)
	for i := uint(0); i < length; i++ {
		value <<= 1
		if (b.Data[b.Offset/8]>>(7-b.Offset%8))&1 > 0 {
			value |= 1
		}
		b.Offset++
	}
	return value
}

type BitUtil struct {
}

func (p *BitUtil) check(number byte, index byte) bool {
	return (number & (1 << index)) != 0
}

func (p *BitUtil) between(number byte, fr byte, to byte) byte {
	return (number >> fr) & ((1 << to - fr) - 1)
}

func (p *BitUtil) to(number byte, to byte) byte {
	return p.between(number, 0, to)
}

func (p *BitUtil) from(number byte, from byte) byte {
	return number >> from
}


type Bitset struct {
	bits []byte
	size int
}

func NewBitset(n int) Bitset {
	return Bitset{make([]byte, (n+7)/8), n}
}

func (bs Bitset) Set(i int) {
    bs.bits[i/8] |= 1 << uint(i%8)
}

func (bs Bitset) Clear(i int) {
    bs.bits[i/8] &= ^(1 << uint(i%8))
}

func (bs Bitset) Get(i int) bool {
    return (bs.bits[i/8] & (1 << uint(i%8))) != 0
}

func (bs Bitset) SetBool(i int, val bool) {
    if val {
        bs.Set(i)
    } else {
        bs.Clear(i)
    }
}

func (bs Bitset) Length() int {
	return bs.size
}


func readValue(buff *bytes.Buffer, length int, signed bool) interface{} {
	var val interface{}

    switch length {
    case 1:
        if signed {
        	var v1 int8
        	binary.Read(buff, binary.BigEndian, &v1)
            val = v1
        } else {
        	var v1 uint8
        	binary.Read(buff, binary.BigEndian, &v1)
        	val = v1
        }
    	return val
    case 2:
        if signed {
        	var v2 int16
            binary.Read(buff, binary.BigEndian, &v2)
        	val = v2
        } else {
        	var v2 uint16
        	binary.Read(buff, binary.BigEndian, &v2)
            val = v2
        }
    	return val
    case 4:
        if signed {
        	var v4 int32
            binary.Read(buff, binary.BigEndian, &v4)
        	val = v4
        } else {
        	var v4 uint32
            binary.Read(buff, binary.BigEndian, &v4)
        	val = v4
        }
    	return val
    default:
        if signed {
        	var v8 int64
            binary.Read(buff, binary.BigEndian, &v8)
        	val = v8
        } else {
        	var v8 uint64
            binary.Read(buff, binary.BigEndian, &v8)
        	val = v8
        }
    	return val
    }
}

func padLeft(str, pad string, lenght int) string {

	if len(str) >= lenght {
		return str
	}

	for {
		str = pad + str
		if len(str) >= lenght {
			return str
		}
	}
}

func crc8_res(buffer []byte) byte {
    crc := byte(0xFF)
    for _, b := range buffer {
        crc ^= b
        for i := 0; i < 8; i++ {
            if (crc & 0x80) != 0 {
                crc = (crc << 1) ^ 0x31
            } else {
                crc = crc << 1
            }
        }
    }
    return crc
}

type HashSet map[string]struct{}

func NewHashSet() HashSet {
    return make(HashSet)
}

func (set HashSet) Add(value string) {
    set[value] = struct{}{}
}

func (set HashSet) Contains(value string) bool {
    _, exists := set[value]
    return exists
}

func (set HashSet) Remove(value string) {
    delete(set, value)
}

func (set HashSet) Size() int {
    return len(set)
}

func (set HashSet) Clear() {
    for k := range set {
        delete(set, k)
    }
}

// func crc_16_rec(pucData []byte, ucLen uint16) byte {
// 	var i int
// 	var ucBit byte
// 	var ucCarry byte

// 	usPoly := 0x8408
// 	var usCRC uint16
	
// 	for i = 0; i < int(ucLen); i++ {
//     	usCRC ^= uint16(pucData[i])
//     	for ucBit = 0; ucBit < 8; ucBit++ {
//         	ucCarry = usCRC & 1
//         	usCRC >>= 1
//         	if len(ucCarry) != 0 {
//             	usCRC ^= usPoly
//         	}
//     	}
//     }

// 	return usCRC
// }