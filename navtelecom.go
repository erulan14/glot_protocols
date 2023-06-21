package main

import (
	"bytes"
	"encoding/binary"
	//"log"
	"net"
	//"unicode/utf8"
	"strings"
	"time"

	//"strconv"
	//"time"
	//"errors"
	//"math"
)

const (
	NAVTELECOM_PROTOCOL = "Navtelecom"
)

type NavtelecomProtocol struct {
}

//var bits Bitset
var bit BitUtil

func checksum(data []byte) byte {
    var c byte
    for _, b := range data {
        c ^= b
    }
    return c
}

var l1 = [...]int{
	4, 5, 6, 7, 8, 29, 30, 31, 32, 45, 46, 47, 48, 49, 50, 51, 52, 56, 63, 64, 65, 69, 72, 78, 79, 80, 81,
    82, 83, 98, 99, 101, 104, 118, 122, 123, 124, 125, 126, 139, 140, 144, 145, 167, 168, 169, 170, 199,
    202, 207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222}

var l2 = [...]int{
	2, 14, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 35, 36, 38, 39, 40, 41, 42, 43, 44, 53, 55, 58,
	59, 60, 61, 62, 66, 68, 71, 75, 100, 106, 108, 110, 111, 112, 113, 114, 115, 116, 117, 119, 120, 121,
	133, 134, 135, 136, 137, 138, 141, 143, 147, 148, 149, 150, 151, 152, 153, 154, 155, 156, 157, 158, 159,
	160, 161, 162, 163, 164, 165, 166, 171, 175, 177, 178, 180, 181, 182, 183, 184, 185, 186, 187, 188, 189,
    190, 191, 192, 200, 201, 223, 224, 225, 226, 227, 228, 229, 230, 231, 232, 233, 234, 235, 236, 237}

var l3 = [...]int{
	84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 142, 146, 198}

var l4 = [...]int{
	1, 3, 9, 10, 11, 12, 13, 15, 16, 33, 34, 37, 54, 57, 67, 74, 76, 102, 103, 105, 127, 128, 129, 130, 131,
	132, 172, 173, 174, 176, 179, 193, 194, 195, 196, 203, 205, 206, 238, 239, 240, 241, 242, 243, 244, 245,
	246, 247, 248, 249, 250, 251, 252}

func (p *NavtelecomProtocol) sendResponse(conn *net.TCPConn, res HandlerResponse, receiver uint32, sender uint32, content *bytes.Buffer) {
	if conn != nil {
   		payload := bytes.NewBuffer(nil)
    	payload.WriteString("@NTC")
    	binary.Write(payload, binary.LittleEndian, sender)
    	binary.Write(payload, binary.LittleEndian, receiver)
    	binary.Write(payload, binary.LittleEndian, uint16(len(content.Bytes())))
    	//log.Println(checksum(content.Bytes()), checksum(payload.Bytes()))
    	payload.WriteByte(checksum(content.Bytes()))
    	payload.WriteByte(checksum(payload.Bytes()))
    
    	for i := 0; i < int(len(content.Bytes())); i++ {
        	payload.WriteByte(content.Bytes()[i])
        }
    
    	//log.Println(payload.Bytes())
    	_, err := conn.Write(payload.Bytes())
		if err != nil {
    		res.error = err
		}
	}
}

func (p *NavtelecomProtocol) handle(readbuff []byte, conn *net.TCPConn, imei string, bits Bitset) HandlerResponse {
	ITEM_LENGTH_MAP := map[int]int{}

	for _, i := range(l1) {
		ITEM_LENGTH_MAP[i] = 1
	}

	for _, i := range(l2) {
		ITEM_LENGTH_MAP[i] = 2
	}

	for _, i := range(l3) {
		ITEM_LENGTH_MAP[i] = 3
	}

	for _, i := range(l4) {
		ITEM_LENGTH_MAP[i] = 4
	}

	ITEM_LENGTH_MAP[70] = 8
	ITEM_LENGTH_MAP[73] = 16
	ITEM_LENGTH_MAP[77] = 37
	ITEM_LENGTH_MAP[94] = 6
	ITEM_LENGTH_MAP[95] = 12
	ITEM_LENGTH_MAP[96] = 24
	ITEM_LENGTH_MAP[97] = 48
	ITEM_LENGTH_MAP[107] = 6
	ITEM_LENGTH_MAP[109] = 6
	ITEM_LENGTH_MAP[197] = 6
	ITEM_LENGTH_MAP[204] = 5
	ITEM_LENGTH_MAP[253] = 8
	ITEM_LENGTH_MAP[254] = 8
	ITEM_LENGTH_MAP[255] = 8


	res := HandlerResponse{}
	res.protocol = NAVTELECOM_PROTOCOL

	buff := bytes.NewBuffer(readbuff)

	//charBytes := make([]byte, utf8.UTFMax)
    //charLen := utf8.EncodeRune(charBytes, '@')
	//idx := bytes.Index(buff.Bytes(), charBytes[:charLen])
	//log.Println("idx", idx, buff.Bytes())

	//var idx string
	idx := string(buff.Bytes()[:4])

	var records []Record

	var gpstime uint32
	var lon int32
	var lat int32
	var alt int32
	var course uint16
	var sat uint8
	var speed float32

	var lon_float float64
	var lat_float float64

	params := make(map[string]interface{})

	if strings.HasPrefix(idx, "@NTC") {
    	buff.Next(4)
    	var receiver uint32
    	var sender uint32
    	var length uint16
    	var csd uint8
    	var csp uint8
    	
    	binary.Read(buff, binary.LittleEndian, &receiver)
    	binary.Read(buff, binary.LittleEndian, &sender)
    	binary.Read(buff, binary.LittleEndian, &length)
    	binary.Read(buff, binary.LittleEndian, &csd)
    	binary.Read(buff, binary.LittleEndian, &csp)
    	
    	var codec string
    	codec = string(buff.Bytes()[:6])
    
    	if strings.HasPrefix(codec, "*>S") {
        	sentence := string(buff.Next(int(length)))
        	//log.Println(sentence)
        	imei := sentence[4:] 
        	res.imei = imei
        	
        	payload := bytes.NewBuffer(nil)
    		payload.WriteString("*<S")
        	p.sendResponse(conn, res, receiver, sender, payload)
        	return res
  
        } else if strings.HasPrefix(codec, "*>FLEX") {
        	res.imei = imei
        
       		buff.Next(6)
        	payload := bytes.NewBuffer(nil)
    		payload.WriteString("*<FLEX")
        	
        	var protocol uint8
    		var protocol_version uint8
    		var struct_version uint8
        	binary.Read(buff, binary.BigEndian, &protocol)
    		binary.Read(buff, binary.BigEndian, &protocol_version)
    		binary.Read(buff, binary.BigEndian, &struct_version)
        	payload.WriteByte(protocol)
        	payload.WriteByte(protocol_version)
        	payload.WriteByte(struct_version)
        
        	var bitCount uint8
        	binary.Read(buff, binary.BigEndian, &bitCount)
        	bits = NewBitset(int(bitCount))
        
        	//log.Println(imei, protocol, protocol_version, struct_version, bitCount, bits)

        	var currentByte uint8
        	for i := 0; i < int(bitCount); i++ {
            	if i % 8 == 0 {
                	var tempByte uint8
                	binary.Read(buff, binary.BigEndian, &tempByte)
                	currentByte = tempByte
            	}
            	bits.SetBool(i, bit.check(currentByte, uint8(7 - i % 8)))
        	}
        
        	res.bits = bits
        
        	p.sendResponse(conn, res, receiver, sender, payload)
        }
    
    } else {
    	res.imei = imei
    	res.bits = bits
    	codec := string(buff.Next(2))
    
    	if strings.HasPrefix(codec, "~A") {
        	// FLEX 1.0 передача накопленных сообщений из черного ящика
        	var count uint8
        	binary.Read(buff, binary.BigEndian, &count)
        	//log.Println(imei, count, bits)
        
        	for i := 0; i < int(count); i++ {
            	for j :=0; j < bits.Length(); j++ {
                	if bits.Get(j) {
                    	//log.Println(j)
                    	switch (j+1) {
                        case 3:
                        	binary.Read(buff, binary.LittleEndian, &gpstime)
                        	break
                        case 8:
                        	var value uint8
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	sat = bit.from(value, 2)
                        	break
                        case 9:
                        	var value uint32
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["fixtime"] = value
                        	break
                        case 10:
                        	binary.Read(buff, binary.LittleEndian, &lat)
                        	lat_float = float64(lat) * 0.0001 / 60
                        	break
                        case 11:
                        	binary.Read(buff, binary.LittleEndian, &lon)
                        	lon_float = float64(lon) * 0.0001 / 60
                        	break
                        case 12:
                        	binary.Read(buff, binary.LittleEndian, &alt)
                        	alt = int32(float32(alt) * 0.1)
                        	break
                        case 13:
                        	binary.Read(buff, binary.LittleEndian, &speed)
                        	break
                        case 14:
                        	binary.Read(buff, binary.LittleEndian, &course)
                        	break
                        case 15:
                        	var value float32
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["odometer"] = value
                        	break
                        case 19:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["power"] = float32(value) * 0.001
                        	break
                        case 20:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["battery"] = float32(value) * 0.001
                        	break
                        case 21:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["ain1"] = float32(value) * 0.001
                        	break
                        case 22:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["ain2"] = float32(value) * 0.001
                        	break
                        case 23:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["ain3"] = float32(value) * 0.001
                        	break
                        case 24:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["ain4"] = float32(value) * 0.001
                        	break
                        case 35:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["afls1"] = value
                        	break
                        case 36:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["afls2"] = value
                        	break
                        case 37:
                        	var value uint32
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["engine_hour_generator"] = value
                        	break
                        case 38:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["fls_485_1"] = value
                        	break
                        case 39:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["fls_485_2"] = value
                        	break
                        case 40:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["fls_485_3"] = value
                        	break
                        case 41:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["fls_485_4"] = value
                        	break
                        case 42:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["fls_485_5"] = value
                        	break
                        case 43:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["fls_485_6"] = value
                        	break
                        case 44:
                        	var value uint16
                        	binary.Read(buff, binary.LittleEndian, &value)
                        	params["fls_232_1"] = value
                        	break
                        
                        default:
                        	buff.Next(ITEM_LENGTH_MAP[j+1])
                        	break
                    	}
                	}
            	}
                 
            	record := Record{
            		int(gpstime),
					int(time.Now().Unix()),
                	Pos{
                    	lat_float,
						lon_float,
						int(alt),
						int(course),
						int(speed),
						int(sat),
                	},
            		params,
				}
				records = append(records, record)
        	}
        
        	res.records = records
        
        	response := bytes.NewBuffer(nil)
    		response.WriteString("~A")
        	response.WriteByte(count)
        	response.WriteByte(crc8_res(response.Bytes()))

        	_, err := conn.Write(response.Bytes())
			if err != nil {
    			res.error = err
			}
   
			return res  	
        } else if strings.HasPrefix(codec, "~T") {
        	// FLEX 1.0 Передача внеочередных сообщений
        
        	var eventindex uint32
        	binary.Read(buff, binary.LittleEndian, &eventindex)
        	
        	for j :=0; j < bits.Length(); j++ {
                if bits.Get(j) {
                    switch (j+1) {
                    case 3:
                        binary.Read(buff, binary.LittleEndian, &gpstime)
                        break
                    case 8:
                       	var value uint8
                        binary.Read(buff, binary.LittleEndian, &value)
                        sat = bit.from(value, 2)
                        break
                    case 9:
                        var value uint32
                        binary.Read(buff, binary.LittleEndian, &value)
                        params["fixtime"] = value
                        break
                    case 10:
                        binary.Read(buff, binary.LittleEndian, &lat)
                        lat_float = float64(lat) * 0.0001 / 60
                        break
                    case 11:
                        binary.Read(buff, binary.LittleEndian, &lon)
                        lon_float = float64(lon) * 0.0001 / 60
                        break
                    case 12:
                        binary.Read(buff, binary.LittleEndian, &alt)
                        alt = int32(float32(alt) * 0.1)
                        break
                    case 13:
                        binary.Read(buff, binary.LittleEndian, &speed)
                        break
                    case 14:
                        binary.Read(buff, binary.LittleEndian, &course)
                        break
                    case 15:
                        var value float32
                        binary.Read(buff, binary.LittleEndian, &value)
                        params["odometer"] = value
                        break
                    case 19:
                        var value uint16
                        binary.Read(buff, binary.LittleEndian, &value)
                        params["power"] = float32(value) * 0.001
                        break
                    case 20:
                        var value uint16
                        binary.Read(buff, binary.LittleEndian, &value)
                        params["battery"] = float32(value) * 0.001
                        break
                    default:
                        buff.Next(ITEM_LENGTH_MAP[j+1])
                        break
                    }
                }
            }
        
        	record := Record{
            	int(gpstime),
				int(time.Now().Unix()),
                Pos{
                    lat_float,
					lon_float,
					int(alt),
					int(course),
					int(speed),
					int(sat),
                },
            	params,
			}
    
			records = append(records, record)
        	res.records = records
        
        	response := bytes.NewBuffer(nil)
    		response.WriteString("~T")
        	binary.Write(response, binary.LittleEndian, eventindex)
        	response.WriteByte(crc8_res(response.Bytes()))

        	_, err := conn.Write(response.Bytes())
			if err != nil {
    			res.error = err
			}
     
			return res
        
        } else if strings.HasPrefix(codec, "~C") {
        	// FLEX 1.0

        	for j :=0; j < bits.Length(); j++ {
                if bits.Get(j) {
                    switch (j+1) {
                    case 3:
                        binary.Read(buff, binary.LittleEndian, &gpstime)
                        break
                    case 8:
                       	var value uint8
                        binary.Read(buff, binary.LittleEndian, &value)
                        sat = bit.from(value, 2)
                        break
                    case 9:
                        var value uint32
                        binary.Read(buff, binary.LittleEndian, &value)
                        params["fixtime"] = value
                        break
                    case 10:
                        binary.Read(buff, binary.LittleEndian, &lat)
                        lat_float = float64(lat) * 0.0001 / 60
                        break
                    case 11:
                        binary.Read(buff, binary.LittleEndian, &lon)
                        lon_float = float64(lon) * 0.0001 / 60
                        break
                    case 12:
                        binary.Read(buff, binary.LittleEndian, &alt)
                        alt = int32(float32(alt) * 0.1)
                        break
                    case 13:
                        binary.Read(buff, binary.LittleEndian, &speed)
                        break
                    case 14:
                        binary.Read(buff, binary.LittleEndian, &course)
                        break
                    case 15:
                        var value float32
                        binary.Read(buff, binary.LittleEndian, &value)
                        params["odometer"] = value
                        break
                    case 19:
                        var value uint16
                        binary.Read(buff, binary.LittleEndian, &value)
                        params["power"] = float32(value) * 0.001
                        break
                    case 20:
                        var value uint16
                        binary.Read(buff, binary.LittleEndian, &value)
                        params["battery"] = float32(value) * 0.001
                        break
                    default:
                        buff.Next(ITEM_LENGTH_MAP[j+1])
                        break
                    }
                }
            }
        
        	record := Record{
            		int(gpstime),
					int(time.Now().Unix()),
                	Pos{
                    	lat_float,
						lon_float,
						int(alt),
						int(course),
						int(speed),
						int(sat),
                	},
            		params,
				}
        
			records = append(records, record)
        	res.records = records
        
        	response := bytes.NewBuffer(nil)
    		response.WriteString("~C")
        	response.WriteByte(crc8_res(response.Bytes()))

        	_, err := conn.Write(response.Bytes())
			if err != nil {
    			res.error = err
			}
     
			return res
        
        } else if strings.HasPrefix(codec, "~E") {
        	// FLEX 2.0
        	var count uint8
        	binary.Read(buff, binary.BigEndian, &count)
        
        	for i := 0; i < int(count); i++ {
            
        	}
        } else if strings.HasPrefix(codec, "~X") {
        	// FLEX 2.0 
        }
    }
	res.error = nil
	return res
}