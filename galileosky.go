package main

import (
	"bytes"
	"encoding/binary"
	//"log"
	"net"
	"strconv"
	"time"

	//"github.com/shimmeringbee/bytecodec/bitbuffer"
	//"github.com/sigurn/crc16"
)

type GalileoskyProtocol struct {
}

const (
	GALILEOSKY_PROTOCOL = "GalileoSky"
)

var gl1 = [...]int{
	0x01, 0x02, 0x35, 0x43, 0xc4, 0xc5, 0xc6, 0xc7,
	0xc8, 0xc9, 0xca, 0xcb, 0xcc, 0xcd, 0xce, 0xcf,
	0xd0, 0xd1, 0xd2, 0xd5, 0x88, 0x89, 0x8a, 0x8b,
	0x8c, 0xa0, 0xaf, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5,
	0xa6, 0xa7, 0xa8, 0xa9, 0xaa, 0xab, 0xac, 0xad,
	0xae}

var gl2 = [...]int{
	0x04, 0x10, 0x34, 0x40, 0x41, 0x42, 0x45, 0x46,
	0x48, 0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56,
	0x57, 0x58, 0x59, 0x60, 0x61, 0x62, 0x70, 0x71,
	0x72, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78, 0x79,
	0x7a, 0x7b, 0x7c, 0x7d, 0xb0, 0xb1, 0xb2, 0xb3,
	0xb4, 0xb5, 0xb6, 0xb7, 0xb8, 0xb9, 0xd6, 0xd7,
	0xd8, 0xd9, 0xda, 0x21}

var gl3 = [...]int{
	0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0x6a,
	0x6b, 0x6c, 0x6d, 0x6e, 0x6f, 0xfa, 0x80, 0x81,
	0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x5d}

var gl4 = [...]int{
	0x20, 0x33, 0x44, 0x90, 0xc0, 0xc1, 0xc2, 0xc3,
	0xd3, 0xd4, 0xdb, 0xdc, 0xdd, 0xde, 0xdf, 0xf0,
	0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7, 0xf8,
	0xf9, 0x5a, 0x47, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5,
	0xf6, 0xf7, 0xf8, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6,
	0xe7, 0xe8, 0xe9}

func bytesToUint24(bytes []byte) uint32 {
	return uint32(bytes[0])<<16 | uint32(bytes[1])<<8 | uint32(bytes[2])
}

func (p *GalileoskyProtocol) sendResponse(conn *net.TCPConn, header int, checksum uint16) {
	if conn != nil {
		reply := bytes.NewBuffer(nil)
		reply.WriteByte(byte(header))
		binary.Write(reply, binary.LittleEndian, uint16(checksum))
		conn.Write(reply.Bytes())
	}
}

func (p *GalileoskyProtocol) decodeTag(params map[string]interface{}, buff *bytes.Buffer, tag int) {
	if tag >= 0x50 && tag <= 0x57 {
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["Prefix_adc"+strconv.Itoa(tag-0x50)] = value
	} else if tag >= 0x60 && tag <= 0x62 {
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["fuel"+strconv.Itoa(tag-0x60)] = value
	} else if tag >= 0xa0 && tag <= 0xaf {
		var value uint8
		binary.Read(buff, binary.BigEndian, &value)
		params["can8BitR"+strconv.Itoa(tag-0xa0+15)] = value
	} else if tag >= 0xb0 && tag <= 0xb9 {
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["can16BitR"+strconv.Itoa(tag-0xb0+5)] = value
	} else if tag >= 0xc4 && tag <= 0xd2 {
		var value uint8
		binary.Read(buff, binary.BigEndian, &value)
		params["can8BitR"+strconv.Itoa(tag-0xc4)] = value
	} else if tag >= 0xd6 && tag <= 0xda {
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["can16BitR"+strconv.Itoa(tag-0xd6)] = value
	} else if tag >= 0xdb && tag <= 0xdf {
		var value uint32
		binary.Read(buff, binary.LittleEndian, &value)
		params["can32BitR"+strconv.Itoa(tag-0xdb)] = value
	} else if tag >= 0xe2 && tag <= 0xe9 {
		var value uint32
		binary.Read(buff, binary.LittleEndian, &value)
		params["userData"+strconv.Itoa(tag-0xe2)] = value
	} else if tag >= 0xf0 && tag <= 0xf9 {
		var value uint32
		binary.Read(buff, binary.LittleEndian, &value)
		params["can32BitR"+strconv.Itoa(tag-0xf0+5)] = value
	} else {
		p.decodeTagOther(params, buff, tag)
	}
}

func (p *GalileoskyProtocol) decodeTagOther(params map[string]interface{}, buff *bytes.Buffer, tag int) {
	ITEM_LENGTH_MAP := make(map[int]int)

	for _, i := range gl1 {
		ITEM_LENGTH_MAP[i] = 1
	}

	for _, i := range gl2 {
		ITEM_LENGTH_MAP[i] = 2
	}

	for _, i := range gl3 {
		ITEM_LENGTH_MAP[i] = 3
	}

	for _, i := range gl4 {
		ITEM_LENGTH_MAP[i] = 4
	}

	ITEM_LENGTH_MAP[0x5b] = 7
	ITEM_LENGTH_MAP[0x5c] = 68
	ITEM_LENGTH_MAP[0xfd] = 8
	ITEM_LENGTH_MAP[0xfe] = 8

	switch tag {
	case 0x01:
		var value uint8
		binary.Read(buff, binary.BigEndian, &value)
		params["version_hw"] = value
		break
	case 0x02:
		var value uint8
		binary.Read(buff, binary.BigEndian, &value)
		params["version_fw"] = value
		break
	case 0x04:
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["deviceId"] = value
		break
	case 0x10:
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["event_id"] = value
		break
	case 0x20:
		var value uint32
		binary.Read(buff, binary.LittleEndian, &value)
		params["gpstime"] = value
		break
	case 0x33:
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["speed"] = float32(value) * 0.1
		binary.Read(buff, binary.LittleEndian, &value)
		params["course"] = float32(value) * 0.1
		break
	case 0x34:
		var value int16
		binary.Read(buff, binary.LittleEndian, &value)
		params["alt"] = value
		break
	case 0x35:
		var value uint8
		binary.Read(buff, binary.BigEndian, &value)
		params["hdop"] = float32(value) * 0.1
		break
	case 0x40:
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["status"] = value
		break
	case 0x41:
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["pwr_ext"] = float32(value) / 1000.0
		break
	case 0x42:
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["pwr_int"] = float32(value) / 1000.0
		break
	case 0x43:
		value, _ := buff.ReadByte()
		params["dev_temp"] = value
		break
	case 0x44:
		var value uint32
		binary.Read(buff, binary.LittleEndian, &value)
		params["accel"] = value
		break
	case 0x45:
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["output"] = value
		break
	case 0x46:
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["input"] = value
		break
	case 0x48:
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["statusExt"] = value
		break
	case 0x58:
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["rs2320"] = value
		break
	case 0x59:
		var value uint16
		binary.Read(buff, binary.LittleEndian, &value)
		params["rs2321"] = value
		break
	case 0x90:
		var value uint32
		binary.Read(buff, binary.LittleEndian, &value)
		params["driver_unq_id"] = value
		break
	case 0xc0:
		var value uint32
		binary.Read(buff, binary.LittleEndian, &value)
		params["fuel_total"] = float32(value) * 0.5
		break
	case 0xc1:
		var value uint8
		binary.Read(buff, binary.BigEndian, &value)
		params["fuel_level"] = float32(value) * 0.4
		binary.Read(buff, binary.BigEndian, &value)
		params["prx_temp1"] = value - 40

		var value2 uint16
		binary.Read(buff, binary.LittleEndian, &value2)
		params["rpm"] = float32(value2) * 0.125
		break
	case 0xc2:
		var value uint32
		binary.Read(buff, binary.LittleEndian, &value)
		params["canB0"] = value
		break
	case 0xc3:
		var value uint32
		binary.Read(buff, binary.LittleEndian, &value)
		params["canB1"] = value
		break
	case 0xd4:
		var value uint32
		binary.Read(buff, binary.LittleEndian, &value)
		params["odometer"] = value
		break
	case 0xe0:
		var value uint32
		binary.Read(buff, binary.LittleEndian, &value)
		params["event_id"] = value
		break
	case 0xe1:
		var value uint8
		binary.Read(buff, binary.BigEndian, &value)
    	params["result"] = string(buff.Next(int(value)))
		break
	case 0xea:
		var value uint8
		binary.Read(buff, binary.BigEndian, &value)
    	params["user_data"] = string(buff.Next(int(value)))
		break
	default:
		buff.Next(ITEM_LENGTH_MAP[tag])
		break
	}
}

func (p *GalileoskyProtocol) handle(readbuff []byte, conn *net.TCPConn, imeis string, bits Bitset) HandlerResponse {
	res := HandlerResponse{}
	res.protocol = GALILEOSKY_PROTOCOL

	buff := bytes.NewBuffer(readbuff)
	temp_buff := bytes.NewBuffer(readbuff)
	//log.Println(buff.Bytes())
	//log.Println(buff.Bytes())

	var header uint8
	binary.Read(buff, binary.BigEndian, &header)
	if header == 0x01 {
		if bytesToUint24(buff.Bytes()[:buff.Cap()-buff.Len()+2]) == 0x01001c {
			records, err, imei := p.decodeIridiumPosition(conn, buff)
        
        	res.imei = imei
			res.records = records
			res.error = err
        	//log.Println(records, err, imei)
        	return res
        
		} else {
			records, err, imei := p.decodePosition(conn, buff, temp_buff)
        
        	res.imei = imei
			res.records = records
			res.error = err
        	//log.Println(records, err, imei)
        	return res

		}
	} else if header == 0x07 {
		// Todo
	} else if header == 0x08 {
    	records, err := p.decodeCompressedPositions(conn, buff)
    	res.imei = imeis
    	res.records = records
    	res.error = err
    	return res
	}

	return res
}

func (p *GalileoskyProtocol) decodeIridiumPosition(conn *net.TCPConn, buff *bytes.Buffer) ([]Record, error, string) {
	var records []Record
	params := make(map[string]interface{})

	var gpstime uint32
	var flags uint8

	var lon_byte uint8
	var lat_byte uint8
	var lon_short uint16
	var lat_short uint16

	var lon_float float64
	var lat_float float64
	var data_length uint16

	buff.Next(9)
	buff.Truncate(15)
	imei := buff.String()

	buff.Next(5)
	binary.Read(buff, binary.BigEndian, &gpstime)
	buff.Next(3)
	binary.Read(buff, binary.BigEndian, &flags)

	binary.Read(buff, binary.BigEndian, &lat_byte)
	binary.Read(buff, binary.BigEndian, &lat_short)
	binary.Read(buff, binary.BigEndian, &lon_byte)
	binary.Read(buff, binary.BigEndian, &lon_short)

	lat_float = float64(lat_byte) + float64(lat_short)/60000.0
	lon_float = float64(lon_byte) + float64(lon_short)/60000.0

	buff.Next(5)
	binary.Read(buff, binary.BigEndian, &data_length)
	//data := bytes.NewBuffer(buff.Bytes()[:data_length])
	//log.Println(data, lon_float, lat_float)

	record := Record{
    	int(gpstime),
		int(time.Now().Unix()),
		Pos{
			lat_float,
			lon_float,
			0,
        	0,
        	0,
			0, //int(params["sat"].(int)),
		},
		params,
	}
	records = append(records, record)

	return records, nil, imei
}

func (p *GalileoskyProtocol) decodePosition(conn *net.TCPConn, buff *bytes.Buffer, temp_buff *bytes.Buffer) ([]Record, error, string) {
	var records []Record
	params := make(map[string]interface{})

	imei := ""
	var endIndex uint16

	var lat int32
	var lon int32
	var lat_float float64
	var lon_float float64

	binary.Read(buff, binary.LittleEndian, &endIndex)
	endIndex = (endIndex & 0x7fff) + uint16(buff.Cap()-buff.Len())
	//table := crc16.MakeTable(crc16.CRC16_MODBUS)
	//checksum := crc16.Checksum(temp_buff.Bytes()[:endIndex], table)

	tags := NewHashSet()
	hasLocation := false

	for uint16(buff.Cap()-buff.Len()) < endIndex {
		var tag uint8
		binary.Read(buff, binary.BigEndian, &tag)

		if tags.Contains(strconv.Itoa(int(tag))) {
			if hasLocation && params["gpstime"] != nil {
				record := Record{
					int(params["gpstime"].(uint32)),
					int(time.Now().Unix()),
					Pos{
						lat_float,
						lon_float,
						int(params["alt"].(int16)),
						int(params["course"].(float32)),
						int(params["speed"].(float32)),
						0, //int(params["sat"].(int)),
					},
					params,
				}
				//log.Println(record)
				records = append(records, record)
			}
			tags.Clear()
			hasLocation = false
		}
		tags.Add(strconv.Itoa(int(tag)))

		if tag == 0x03 {
			imei = string(buff.Next(15))
		} else if tag == 0x30 {
			hasLocation = true
			buff.Next(1)
			binary.Read(buff, binary.LittleEndian, &lat)
			binary.Read(buff, binary.LittleEndian, &lon)
			lat_float = float64(lat) / 1000000.0
			lon_float = float64(lon) / 1000000.0

			//log.Println(lat_float, lon_float)
		} else {
			p.decodeTag(params, buff, int(tag))
		}

	}

	if hasLocation && params["gpstime"] != nil {
		record := Record{
			int(params["gpstime"].(uint32)),
			int(time.Now().Unix()),
			Pos{
				lat_float,
				lon_float,
				int(params["alt"].(int16)),
				int(params["course"].(float32)),
				int(params["speed"].(float32)),
				0, //int(params["sat"].(int)),
			},
			params,
		}
		records = append(records, record)
    }
	// } else if _, ok := params["result"]; ok {
	// 	record := Record{
	// 		int(params["gpstime"].(uint32)),
	// 		int(time.Now().Unix()),
	// 		Pos{
	// 			lat_float,
	// 			lon_float,
	// 			int(params["alt"].(int16)),
	// 			int(params["course"].(float32)),
	// 			int(params["speed"].(float32)),
	// 			0, //int(params["sat"].(int)),
	// 		},
	// 		params,
	// 	}
	// 	records = append(records, record)
	// }

	var checksum1 uint16
	binary.Read(buff, binary.LittleEndian, &checksum1)
	//log.Println(checksum, checksum1)

	p.sendResponse(conn, 0x02, checksum1)
	return records, nil, imei
}

func (p *GalileoskyProtocol) decodeMinimalDataSet(params map[string]interface{}, buff *bytes.Buffer) {
	bits := NewBitBuffer(buff.Next(10))
	bits.ReadUnsigned(27)

	params["min_data_lon"] = float64((360*float32(bits.ReadUnsigned(22))/4194304.0) - 180)
	params["min_data_lat"] = float64((180*float32(bits.ReadUnsigned(21))/2097152.0) - 90)

	alarm := bits.ReadUnsigned(1)
	if alarm > 0 {
		params["alarm"] = "ALARM_GENERAL"
	}
}

func (p *GalileoskyProtocol) decodeCompressedPositions(conn *net.TCPConn, buff *bytes.Buffer) ([]Record, error) {
	var records []Record
	params := make(map[string]interface{})

	var lat int32
	var lon int32
	var lat_float float64
	var lon_float float64

	var length uint16
	binary.Read(buff, binary.LittleEndian, &length)
	length = (length & 0x7fff) + uint16(buff.Cap()-buff.Len())
	//log.Println(length)

	is_start_tags := false
	hasLocation := false
	var tags []int

	for uint16(buff.Cap()-buff.Len()) < length {
    	//p.decodeMinimalDataSet(params, buff)
    	buff.Next(10)
    	
    	if is_start_tags == false {
        	var abyte uint8
			binary.Read(buff, binary.BigEndian, &abyte)
        	
        	tags = make([]int, int(abyte & 0x7f))
			for i := 0; i < int(len(tags)); i++ {
                var value uint8
				binary.Read(buff, binary.BigEndian, &value)
            	tags[i] = int(value)
            }

        	is_start_tags = true
    	}
   
		for i := 0; i < int(len(tags)); i++ {
        	if tags[i] == 0x30 {
				hasLocation = true
				buff.Next(1)
				binary.Read(buff, binary.LittleEndian, &lat)
				binary.Read(buff, binary.LittleEndian, &lon)
				lat_float = float64(lat) / 1000000.0
				lon_float = float64(lon) / 1000000.0
            	//log.Println(lat_float, lon_float)
            } else {
				p.decodeTag(params, buff, tags[i])
            }
		}
    	
    	if hasLocation {
        	record := Record{
				int(params["gpstime"].(uint32)),
				int(time.Now().Unix()),
				Pos{
					lat_float,
					lon_float,
					int(params["alt"].(int16)),
					int(params["course"].(float32)),
					int(params["speed"].(float32)),
					//int(params["sat"].(int)),
            		0,
				},
				params,
			}
			records = append(records, record)
        } else {
        	record := Record{
				int(params["gpstime"].(uint32)),
				int(time.Now().Unix()),
				Pos{
					params["lat"].(float64),
					params["lon"].(float64),
					int(params["alt"].(int16)),
                    int(params["course"].(float32)),
                	int(params["speed"].(float32)),
					//int(params["sat"].(int)),
            		0,
				},
				params,
			}
			records = append(records, record)
        }
	}
	//log.Println(records)
	
	var checksum uint16
	//log.Println(buff.Bytes())
	binary.Read(buff, binary.LittleEndian, &checksum)
	p.sendResponse(conn, 0x02, checksum)

	return records, nil
}
