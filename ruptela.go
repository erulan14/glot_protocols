// 	/**
//   	- Ruptela Protocol
//   	- Version 1.0
//   	- Protocol v1.0 && v1.1
//  */
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	//"log"
	"net"
	"strconv"
	"time"
	"github.com/sigurn/crc16"
)

const (
	MSG_RECORDS 				= 0x01
	MSG_DEVICE_CONFIGURATION	= 0x02
	MSG_DEVICE_VERSION 			= 0x03
	MSG_FIRMWARE_UPDATE 		= 0x04
	MSG_SET_CONNECTION			= 0x05
	MSG_SET_ODOMETER 			= 0x06
	MSG_SMS_VIA_GPRS_RESPONSE 	= 0x07
	MSG_SMS_VIA_GPRS 			= 0x08
	MSG_DTCS 					= 0x09
	MSG_IDENTIFICATION			= 0x0F
	MSG_SET_IO 					= 0x11
	MSG_FILES					= 0x25
	MSG_RECORDS_EXT 			= 0x44
	RUPTELA_PROTOCOL        	= "Ruptela"
)

type RuptelaProtocol struct {
}

func (p *RuptelaProtocol) handle(readbuff []byte, conn *net.TCPConn, imei string, bits Bitset) HandlerResponse {

	res := HandlerResponse{}
	res.protocol = RUPTELA_PROTOCOL

	buff := bytes.NewBuffer(readbuff)

	records, err1, imei := p.getRecords(buff)
	if err1 != nil {
    	if err1 == errors.New("crc16") {
        	_, err2 := conn.Write([]byte{0x00, 0x02, 0x00, 0x00, 0x13, 0xbc})
			if err2 != nil {
				res.error = err2
			}
        }
    	
		res.error = err1
    	return res
	}

	res.records = records
	res.imei = imei

	// ACK
	_, err2 := conn.Write([]byte{0x00, 0x02, 0x64, 0x01, 0x13, 0xbc})
	if err2 != nil {
		res.error = err2
	}

	return res
}

// func decodeCommandResponse(records []Record, tip int, buff *bytes.Buffer) ([]Record, error, string) {
// 	switch (tip) {
//     case MSG_DEVICE_CONFIGURATION:
//     case MSG_DEVICE_VERSION:
//     case MSG_FIRMWARE_UPDATE:
//     case MSG_SMS_VIA_GPRS_RESPONSE:
//     case MSG_SET_IO:
//     	return records, nil, ""
//     default:
//     	return nil, errors.New("Unknown type requests"), ""
// 	}
// }

func (p *RuptelaProtocol) decodeParameter(params map[string]interface{}, id int, buff *bytes.Buffer, length int) {
	switch id {
    // case 2:
    // case 3:
    // case 4:
    // 	params["din" + strconv.Itoa(id - 1)] = readValue(buff, length, false)
    // 	break
    case 5:
    	params["ign"] = readValue(buff, length, false)
    	break
    case 29:
    	params["pwr_ext"] = readValue(buff, length, false)
    	break
    case 30:
    	iAreaId, _ := readValue(buff, length, false).(float32)
    	params["pwr_int"] = iAreaId * 0.001
    	break
    case 32:
    	params["pcb_t"] = readValue(buff, length, true)
    	break
    case 65:
    	params["odometer"] = readValue(buff, length, false)
    	break
    	
    case 74:
    	iAreaId, _ := readValue(buff, length, true).(float32)
    	params["t3"] = iAreaId * 0.1
    	break
    // case 78:
    // case 79:
    // case 80:
    // 	iAreaId, _ := readValue(buff, length, true).(float32)
    // 	params["t" + strconv.Itoa(id - 78)] = iAreaId * 0.1
    // 	break
    case 114:
    	params["can_distance"] = readValue(buff, length, false)
    	break
    case 115:
    	params["can_engine_temp_coolant"] = readValue(buff, length, false)
    	break
    case 134:
    	iAreaId, _ := readValue(buff, length, false).(int)
    	if iAreaId > 0 {
        	params["sos"] = "ALARM_BRAKING"
    	}
    	break
    case 136:
    	iAreaId, _ := readValue(buff, length, false).(int)
    	if iAreaId > 0 {
        	params["sos"] = "ALARM_ACCELERATION"
    	}
    	break
    case 197:
    	iAreaId, _ := readValue(buff, length, false).(float32)
    	params["can_engine_speed"] = iAreaId * 0.125
    	break
    case 203:
    	params["can_engine_hours"] = readValue(buff, length, false)
    	break
    case 207:
    	params["fuel_level1"] = readValue(buff, length, false)
    	break
    default:
    	params["io_"+strconv.Itoa(length)+"_"+strconv.Itoa(id)] = readValue(buff, length, false)
    	break
	}
}

func (p *RuptelaProtocol) getRecords(buff *bytes.Buffer) ([]Record, error, string) {

	var records []Record

	var packet_length uint16
	var imei int64
	var tip uint8           // tip zahteva
	var records_left uint8  // broj preostalih recorda na uređaju (ne koristimo za sada)
	var records_count uint8 // broj recorda u tekućem zahtevu
	var gpstime uint32
	var lon int32
	var lat int32
	var alt uint16
	var course uint16
	var sat uint8
	var speed uint16
	var hdop uint8
	var crc16_check uint16

	params := make(map[string]interface{})

	binary.Read(buff, binary.BigEndian, &packet_length)

	if packet_length > uint16(buff.Len()) {
    	return nil, errors.New("Data length big"), ""
	}

	crc_buffer := bytes.NewBuffer(buff.Bytes()[:packet_length])
	binary.Read(buff, binary.BigEndian, &imei)
	binary.Read(buff, binary.BigEndian, &tip)

	imeiString := padLeft(strconv.FormatInt(imei, 10), "0", 15)
	//log.Println("INFO", "Tip:", tip)

	if tip == MSG_RECORDS || tip == MSG_RECORDS_EXT {
    	binary.Read(buff, binary.BigEndian, &records_left)
		binary.Read(buff, binary.BigEndian, &records_count)
    
    	for i := 0; i < int(records_count); i++ {
        	binary.Read(buff, binary.BigEndian, &gpstime)
        	buff.Next(1)
        	if tip == MSG_RECORDS_EXT {
            	buff.Next(1)
            } 
        	buff.Next(1)
        
        	binary.Read(buff, binary.BigEndian, &lon)
			binary.Read(buff, binary.BigEndian, &lat)
			binary.Read(buff, binary.BigEndian, &alt)
			binary.Read(buff, binary.BigEndian, &course)
			binary.Read(buff, binary.BigEndian, &sat)
			binary.Read(buff, binary.BigEndian, &speed)
        
        	lon_float := float64(lon) / 10000000
			lat_float := float64(lat) / 10000000
        
        	binary.Read(buff, binary.BigEndian, &hdop)
        	hdop_float := float32(hdop) / 10.0
        	params["hdop"] = hdop_float
        
        	pos := Pos{
            	lat_float,
            	lon_float,
            	int(float32(alt) / 10.0),
            	int(float32(course) / 100.0),
            	int(speed),
            	int(sat),
        	}
        
        	if tip == MSG_RECORDS_EXT {
            	var event_io uint16
            	binary.Read(buff, binary.BigEndian, &event_io)
            	params["event_io"] = event_io
            } else {
            	var event_io uint8
            	binary.Read(buff, binary.BigEndian, &event_io)
            	params["event_io"] = event_io
            }
        
        	var bytes_count uint8
        	
        	// Read 1 byte data
        	binary.Read(buff, binary.BigEndian, &bytes_count)
        	for j := 0; j < int(bytes_count); j++ {
            	if tip == MSG_RECORDS_EXT {
                	var id uint16
                	binary.Read(buff, binary.BigEndian, &id)
                	p.decodeParameter(params, int(id), buff, 1)
                } else {
                	var id uint8
                	binary.Read(buff, binary.BigEndian, &id)
                	p.decodeParameter(params, int(id), buff, 1)
                }
            }
        
        	// Read 2 byte data
        	binary.Read(buff, binary.BigEndian, &bytes_count)
        	for j := 0; j < int(bytes_count); j++ {
            	if tip == MSG_RECORDS_EXT {
                	var id uint16
                	binary.Read(buff, binary.BigEndian, &id)
                	p.decodeParameter(params, int(id), buff, 2)
                } else {
                	var id uint8
                	binary.Read(buff, binary.BigEndian, &id)
                	p.decodeParameter(params, int(id), buff, 2)
                }
            }
        
        	// Read 4 byte data
        	binary.Read(buff, binary.BigEndian, &bytes_count)
        	for j := 0; j < int(bytes_count); j++ {
            	if tip == MSG_RECORDS_EXT {
                	var id uint16
                	binary.Read(buff, binary.BigEndian, &id)
                	p.decodeParameter(params, int(id), buff, 4)
                } else {
                	var id uint8
                	binary.Read(buff, binary.BigEndian, &id)
                	p.decodeParameter(params, int(id), buff, 4)
                }
            }
        
        	// Read 8 byte data
        	binary.Read(buff, binary.BigEndian, &bytes_count)
        	for j := 0; j < int(bytes_count); j++ {
            	if tip == MSG_RECORDS_EXT {
                	var id uint16
                	binary.Read(buff, binary.BigEndian, &id)
                	p.decodeParameter(params, int(id), buff, 8)
                } else {
                	var id uint8
                	binary.Read(buff, binary.BigEndian, &id)
                	p.decodeParameter(params, int(id), buff, 8)
                }
            }
        
        	record := Record{
            	int(gpstime),
        		int(time.Now().Unix()),
            	pos,
            	params,
            }
        
        	records = append(records, record)
        }
    
    	binary.Read(buff, binary.BigEndian, &crc16_check)
		table := crc16.MakeTable(crc16.CRC16_KERMIT)
		crc := crc16.Checksum(crc_buffer.Bytes(), table)
    
    	if crc != crc16_check {
    		return records, errors.New("crc16"), imeiString
		}
    	
    	return records, nil, imeiString
    } else if tip == MSG_DTCS {
    	// TODO
    } else if tip == MSG_FILES {
    	// TODO
    } else if tip == MSG_IDENTIFICATION {
    	// TODO
    } 
	return nil, errors.New("Unknown type requests"), ""
}