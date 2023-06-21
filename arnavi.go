package main

import (
	//"log"
	"encoding/binary"
	"bytes"
	"net"
	"time"
	"strconv"
)

type ArnaviProtocol struct {
}

const (
	ARNAVI_PROTOCOL = "Arnavi"

	HEADER_START_SIGN = 0xff
	HEADER_VERSION_1 = 0x22
	HEADER_VERSION_2 = 0x23

	RECORD_PING = 0x00
	RECORD_DATA = 0x01
	RECORD_TEXT = 0x03
	RECORD_FILE = 0x04
	RECORD_BINARY = 0x06
	
	TAG_LATITUDE = 3
	TAG_LONGITUDE = 4
	TAG_COORD_PARAMS = 5
)

func (p *ArnaviProtocol) sendResponse(res HandlerResponse, conn *net.TCPConn, version byte, index uint8) {
	if conn != nil {
    	response := bytes.NewBuffer(nil)
    	response.WriteByte(0x7b)
    	if version == HEADER_VERSION_1 {
        	response.WriteByte(0x00)
        	response.WriteByte(byte(index))
        } else if version == HEADER_VERSION_2 {
        	response.WriteByte(0x04)
        	response.WriteByte(0x00)
        
        	timestamp := time.Now().Unix()

    		// Encode timestamp as 4-byte little-endian integer
    		timeBytes := make([]byte, 4)
    		binary.LittleEndian.PutUint32(timeBytes, uint32(timestamp))
        
        	checksum := binary.BigEndian.Uint16(timeBytes)
    		checksumMod256 := byte(checksum % 256)
        	response.WriteByte(checksumMod256)
        
        	for i := 0; i < int(len(timeBytes)); i++ {
        		response.WriteByte(timeBytes[i])
        	}
        }
    	response.WriteByte(0x7d)
    	_, err := conn.Write(response.Bytes())
		if err != nil {
    		res.error = err
		}
	}
}

func (p *ArnaviProtocol) decodePosition(buff *bytes.Buffer, length uint16, gpstime uint32) []Record {
	var records []Record
	params := make(map[string]interface{})
	var lon_float float32
	var lat_float float32
	var alt uint8
	var course uint8
	var sat byte
	var speed uint8

	readBytes := 0
	for readBytes < int(length) {
    	var tag uint8
    	binary.Read(buff, binary.BigEndian, &tag)
    
    	switch (tag) {
        case TAG_LATITUDE:
        	binary.Read(buff, binary.LittleEndian, &lat_float)
        	break
        case TAG_LONGITUDE:
        	binary.Read(buff, binary.LittleEndian, &lon_float)
        	break
        case TAG_COORD_PARAMS:
        	binary.Read(buff, binary.LittleEndian, &course)
        	binary.Read(buff, binary.LittleEndian, &alt)
        	
        	binary.Read(buff, binary.LittleEndian, &sat)
        	
        
        	binary.Read(buff, binary.LittleEndian, &speed)
        	break
        default:
        	buff.Next(5)
        	break
    	}
    
    	record := Record{
        	int(gpstime),
        	int(time.Now().Unix()),
        	Pos{
            	float64(lat_float),
            	float64(lon_float),
            	int(alt) * 10,
            	int(course) * 2,
            	int(speed),
            	int(sat & 0x0f + (sat >> 4) & 0x0f),
            },
        	params,
        }
    	records = append(records, record)
    
    	readBytes += 1 + 4
	}
	return records
}

func (p *ArnaviProtocol) handle(readbuff []byte, conn *net.TCPConn, imei string, bits Bitset) HandlerResponse {
	var gpstime uint32

	res := HandlerResponse{}
	res.protocol = ARNAVI_PROTOCOL

	buff := bytes.NewBuffer(readbuff)

	var startSign byte
	binary.Read(buff, binary.BigEndian, &startSign)
	
	if startSign == HEADER_START_SIGN {
    	var version byte
    	var imei int64
    
    	binary.Read(buff, binary.BigEndian, &version)
    	binary.Read(buff, binary.LittleEndian, &imei)
    
    	imeiString := padLeft(strconv.FormatInt(imei, 10), "0", 15)
    	res.imei = imeiString
     	
    	return res
	}

	var index uint8
	binary.Read(buff, binary.BigEndian, &index)
	var recordType byte
	binary.Read(buff, binary.BigEndian, &recordType)
	
	// TODO While bytes
	for buff.Len() > 0 {
    	switch (recordType) {
        case RECORD_PING:
        	break
        case RECORD_DATA:
        	break
        case RECORD_TEXT:
        	break
        case RECORD_FILE:
        	break
        case RECORD_BINARY:
        	var length uint16
        	binary.Read(buff, binary.LittleEndian, &length)
        	binary.Read(buff, binary.LittleEndian, &gpstime)
        
        	if recordType == RECORD_DATA {
            	res.records = p.decodePosition(buff, length, gpstime)
            	res.imei = imei
            } else {
            	buff.Next(int(length))
            }
        
        	buff.Next(1)
        	break
        default:
        	return res
    	}
    	
    	binary.Read(buff, binary.BigEndian, &recordType)
	}

	p.sendResponse(res, conn, HEADER_VERSION_1, index)

	return res
}