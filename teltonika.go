// 	/**
//   	- Teltonika Protocol
//   	- Version 1.0
//   	- Protocol CODEC_8, CODEC_8_EXT
//  */

package main

import (
	// "fmt"
	"bytes"
	"encoding/binary"
	//"log"
	"net"
	"time"

	"strconv"
	"errors"
	"github.com/sigurn/crc16"
	//"github.com/howeyc/crc16"
)

const (
	TELTONIKA_PROTOCOL     = "teltonika"
	TELTONIKA_CODEC_GH3000 = 0x07
	TELTONIKA_CODEC_8      = 0x08
	TELTONIKA_CODEC_8_EXT  = 0x8E
	TELTONIKA_CODEC_12     = 0x0C
	TELTONIKA_CODEC_13     = 0x0D
	TELTONIKA_CODEC_14	   = 0x0E
	TELTONIKA_CODEC_16     = 0x10
)

type TeltonikaProtocol struct {
}

func (p *TeltonikaProtocol) handle(readbuff []byte, conn *net.TCPConn, imei string, bits Bitset) HandlerResponse {

	buff := bytes.NewBuffer(readbuff)

	var start_bytes uint16

	binary.Read(buff, binary.BigEndian, &start_bytes)
	//log.Println("INFO", "start_bytes", start_bytes)

	res := HandlerResponse{}
	res.protocol = TELTONIKA_PROTOCOL

	// Если у нас есть что-то в первых 2 байтах, то устройство отправило свой IMEI
	if start_bytes > 0 && start_bytes < 16 {

		imei, err1 := p.getIMEI(buff)
		if err1 != nil {
			res.error = err1
		}

		res.imei = imei

		//log.Println("INFO", "Device IMEI:", res.imei)

		_, err2 := conn.Write([]byte{0x01}) // ACK
		if err2 != nil {
			res.error = err2
		}
		// Если первые два байта равны нулю, то устройство отправило записи GPS.
	} else {

		res.imei = imei

		records, err1 := p.getRecords(buff)
		if err1 != nil {
        	if err1 == errors.New("crc16") {
            	err2 := binary.Write(conn, binary.BigEndian, int32(0))
				if err2 != nil {
					res.error = err2
				}
        	}
        
			res.error = err1
        	return res
		}
		res.records = records

    	
		err2 := binary.Write(conn, binary.BigEndian, int32(len(records)))
		if err2 != nil {
			res.error = err2
		}
	}

	return res
}

func (p *TeltonikaProtocol) getIMEI(buff *bytes.Buffer) (string, error) {

	var imei string

	buff.Truncate(15)

	imei = buff.String()

	if imei == "" {
		return "", errors.New("Imei is nil")
	}

	return padLeft(imei, "0", 15), nil
}

func (p *TeltonikaProtocol) decodeParameter(params map[string]interface{}, id int, buff *bytes.Buffer, length int) {
	switch id {
    case 1:
    	params["din1"] = readValue(buff, length, false)
    	break
    case 2:
    	params["din2"] = readValue(buff, length, false)
    	break
    case 3:
    	params["din3"] = readValue(buff, length, false)
    	break
    case 262:
    	params["din4"] = readValue(buff, length, false)
    	break
    case 179:
    	params["dout1"] = readValue(buff, length, false)
    	break
    case 180:
    	params["dout2"] = readValue(buff, length, false)
    	break
    case 16:
    	params["odometer"] = readValue(buff, length, false)
    	break
    case 24:
    	params["speed"] = readValue(buff, length, false)
    	break
    case 66:
    	params["pwr_ext"] = readValue(buff, length, false)
    	break
    case 67:
    	params["pwr_int"] = readValue(buff, length, false)
    	break
    case 239:
    	params["ign"] = readValue(buff, length, false)
    	break
    case 240:
    	params["mov"] = readValue(buff, length, false)
    	break
    default:
    	params["io_"+strconv.Itoa(id)] = readValue(buff, length, false)
    	break
	}
}

func (p *TeltonikaProtocol) readExtByte(buff *bytes.Buffer, codec int, codecs ...int) interface{} {
	ext := false
	for _, i := range(codecs) {
    	if codec == i {
        	ext = true
        	break
    	}
	}

	if ext {
    	var res uint16
    	binary.Read(buff, binary.BigEndian, &res)
    	return res
    } else {
    	var res uint8
    	binary.Read(buff, binary.BigEndian, &res)
    	return res
    }
}

func (p *TeltonikaProtocol) getRecords(buff *bytes.Buffer) ([]Record, error) {
	var records []Record

	var data_length uint32
	var codec uint8
	var priority uint8      // мы пока не используем
	var records_count uint8 // номер записи в текущем запросе
	var gpstime uint64
	var lon int32
	var lat int32
	var alt int16
	var course uint16
	var sat uint8
	var speed uint16
	var crc16_check uint32

	params := make(map[string]interface{})

	buff.Next(2)

	binary.Read(buff, binary.BigEndian, &data_length)

	if data_length > uint32(buff.Len()) {
		return nil, errors.New("Data length big")
	}

	crc_buffer := bytes.NewBuffer(buff.Bytes()[:data_length])
	binary.Read(buff, binary.BigEndian, &codec)

	if codec == TELTONIKA_CODEC_12 {
		// TODO ?
		return nil, errors.New("CODEC 12")
	}

	binary.Read(buff, binary.BigEndian, &records_count)

	if records_count == 0 {
		return nil, errors.New("no records")
	}

	//log.Println("INFO", "Number of records in the request:", records_count)
	//log.Println("INFO", "Codec:", codec)

	for i := 0; i < int(records_count); i++ {

		if codec == TELTONIKA_CODEC_GH3000 {
			// TODO
		} else {
        	// GPS ELEMENT
			binary.Read(buff, binary.BigEndian, &gpstime)
			binary.Read(buff, binary.BigEndian, &priority)
			binary.Read(buff, binary.BigEndian, &lon)
			binary.Read(buff, binary.BigEndian, &lat)
			binary.Read(buff, binary.BigEndian, &alt)
			binary.Read(buff, binary.BigEndian, &course)
			binary.Read(buff, binary.BigEndian, &sat)
			binary.Read(buff, binary.BigEndian, &speed)

			lon_float := float64(lon) / 10000000
			lat_float := float64(lat) / 10000000
        
        	pos := Pos{
            	lat_float,
				lon_float,
				int(alt),
				int(course),
				int(speed),
				int(sat),
            }
        
        	if codec == TELTONIKA_CODEC_8 || codec == TELTONIKA_CODEC_8_EXT {
            	
            	if codec == TELTONIKA_CODEC_8_EXT {
                	var event_io uint16
                	var n_total uint16
                	var bytes_count uint16
                	var id uint16
                	var length uint16
                
                	binary.Read(buff, binary.BigEndian, &event_io)
        			binary.Read(buff, binary.BigEndian, &n_total)
                
                	if int(event_io) > 0	{
            			params["event_io"] = event_io
                	}
                
                	binary.Read(buff, binary.BigEndian, &bytes_count)
        
        			for j := 0; j < int(bytes_count); j++ {
                		binary.Read(buff, binary.BigEndian, &id)
                		p.decodeParameter(params, int(id), buff, 1)
        			}
        
        			binary.Read(buff, binary.BigEndian, &bytes_count)
        
        			for  j := 0; j < int(bytes_count); j++ {
                		binary.Read(buff, binary.BigEndian, &id)
                		p.decodeParameter(params, int(id), buff, 2)
        			}
        
        			binary.Read(buff, binary.BigEndian, &bytes_count)
        
        			for  j := 0; j < int(bytes_count); j++ {
                		binary.Read(buff, binary.BigEndian, &id)
                		p.decodeParameter(params, int(id), buff, 4)
        			}
        
        			binary.Read(buff, binary.BigEndian, &bytes_count)
        
        			for  j := 0; j < int(bytes_count); j++ {
                		binary.Read(buff, binary.BigEndian, &id)
                		p.decodeParameter(params, int(id), buff, 8)
        			}
                
                	binary.Read(buff, binary.BigEndian, &bytes_count)
                
                	for  j := 0; j < int(bytes_count); j++ {
                		binary.Read(buff, binary.BigEndian, &id)
                    	binary.Read(buff, binary.BigEndian, &length)
                    	
                    	if id == 256 {
                        	buff.Truncate(int(length))
                        } else if id == 281 {
                        	buff.Truncate(int(length))
                        } else if id == 385 {
                        	// TODO
                        	buff.Truncate(int(length))
                        } else {
                        	buff.Truncate(int(length))
                        }
        			}
                
                } else {
                	var event_io uint8
                	var n_total uint8
                	var bytes_count uint8
                	var id uint8
                	
                	binary.Read(buff, binary.BigEndian, &event_io)
        			binary.Read(buff, binary.BigEndian, &n_total)
                
                	if int(event_io) > 0	{
            			params["event_io"] = event_io
                	}
                
                	binary.Read(buff, binary.BigEndian, &bytes_count)
        
        			for j := 0; j < int(bytes_count); j++ {
                		binary.Read(buff, binary.BigEndian, &id)
                		p.decodeParameter(params, int(id), buff, 1)
        			}
        
        			binary.Read(buff, binary.BigEndian, &bytes_count)
        
        			for  j := 0; j < int(bytes_count); j++ {
                		binary.Read(buff, binary.BigEndian, &id)
                		p.decodeParameter(params, int(id), buff, 2)
        			}
        
        			binary.Read(buff, binary.BigEndian, &bytes_count)
        
        			for  j := 0; j < int(bytes_count); j++ {
                		binary.Read(buff, binary.BigEndian, &id)
                		p.decodeParameter(params, int(id), buff, 4)
        			}
        
        			binary.Read(buff, binary.BigEndian, &bytes_count)
        
        			for  j := 0; j < int(bytes_count); j++ {
                		binary.Read(buff, binary.BigEndian, &id)
                		p.decodeParameter(params, int(id), buff, 8)
        			}
                }
            
            	record := Record{
            		int(gpstime / 1000),
					int(time.Now().Unix()),
					pos,
            		params,
				}
				records = append(records, record)
        	}
		}
	}

	buff.Next(1)

	binary.Read(buff, binary.BigEndian, &crc16_check)

	table := crc16.MakeTable(crc16.CRC16_ARC)
	crc := crc16.Checksum(crc_buffer.Bytes(), table)

	if int(crc) != int(crc16_check) {
    	//log.Println(data_length, codec, records)
    	return records, errors.New("crc16")
	}

	return records, nil
}
