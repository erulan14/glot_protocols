package main

import (
	"log"
	"encoding/binary"
	"bytes"
	"net"
	"strconv"
	"time"
	"github.com/sigurn/crc16"
	"github.com/LdDl/go-egts/crc/crc8"
)

type EgtsProtocol struct {
}

const (
	EGTS_PROTOCOL = "EGTS"
	PT_RESPONSE = 0
	PT_APPDATA = 1
	PT_SIGNED_APPDATA = 2

	SERVICE_AUTH = 1
	SERVICE_TELEDATA = 2
	SERVICE_COMMANDS = 4
	SERVICE_FIRMWARE = 9
	SERVICE_ECALL = 10

	MSG_RECORD_RESPONSE = 0
	MSG_TERM_IDENTITY = 1
	MSG_MODULE_DATA = 1
	MSG_VEHICLE_DATA = 3
	MSG_AUTH_PARAMS = 4
	MSG_AUTH_INFO = 5
	MSG_SERVICE_INFO = 6
	MSG_RESULT_CODE = 7
	MSG_POS_DATA = 16
	MSG_EXT_POS_DATA = 17
	MSG_AD_SENSORS_DATA = 18
	MSG_COUNTERS_DATA = 19
	MSG_STATE_DATA = 20
	MSG_LOOPIN_DATA = 22
	MSG_ABS_DIG_SENS_DATA = 23
	MSG_ABS_AN_SENS_DATA = 24
	MSG_ABS_CNTR_DATA = 25
	MSG_ABS_LOOPIN_DATA = 26
	MSG_LIQUID_LEVEL_SENSOR = 27 
	MSG_PASSENGERS_COUNTERS = 28
)

var packetId int

func readUnsignedMediumLE(data []byte, offset int) uint32 {
    b1 := uint32(data[offset] & 0xff)
    b2 := uint32(data[offset+1] & 0xff)
    b3 := uint32(data[offset+2] & 0xff)
    return (b3 << 16) | (b2 << 8) | b1
}

func readMediumLE(data []byte, offset int) int32 {
    b1 := int32(data[offset])
    b2 := int32(data[offset+1])
    b3 := int32(data[offset+2])
    return (b3 << 16) | (b2 << 8) | b1
}

func (p *EgtsProtocol) sendResponse(res HandlerResponse, conn *net.TCPConn, packetType int, index int, serviceType int, typee int, content *bytes.Buffer) {
	if conn != nil {
    	data := bytes.NewBuffer(nil)
    	data.WriteByte(byte(typee))
    	
    	binary.Write(data, binary.LittleEndian, uint16(content.Len()))
    	for i := 0; i < int(len(content.Bytes())); i++ {
        	data.WriteByte(content.Bytes()[i])
        }
    
    	record := bytes.NewBuffer(nil)
    	if packetType == PT_RESPONSE {
        	binary.Write(record, binary.LittleEndian, uint16(index))
        	record.WriteByte(0)
    	}
    	binary.Write(record, binary.LittleEndian, uint16(data.Len()))
    	binary.Write(record, binary.LittleEndian, uint16(0))
    	record.WriteByte(0)
    	record.WriteByte(byte(serviceType))
    	record.WriteByte(byte(serviceType))
    	for i := 0; i < int(len(data.Bytes())); i++ {
        	record.WriteByte(data.Bytes()[i])
        }
    	// TODO crc16
    
    	table := crc16.MakeTable(crc16.CRC16_CCITT_FALSE)
		recordChecksum := crc16.Checksum(record.Bytes(), table)
    
		// if recordChecksum != crc16_check {
		// return records, errors.New("crc16"), imeiString
		// }
    
    	response := bytes.NewBuffer(nil)
    	response.WriteByte(1)
    	response.WriteByte(0)
    	response.WriteByte(0)
    	response.WriteByte(5 + 2 + 2 + 2)
    	response.WriteByte(0)
    	packetId++
    	binary.Write(response, binary.LittleEndian, uint16(record.Len()))
    	binary.Write(response, binary.LittleEndian, uint16(packetId))
    	response.WriteByte(byte(packetType))
    	// TODO crc8
    	
    	table2 := crc8.MakeTable(crc8.Params{
			Poly:   0x31,
			Init:   0xFF,
			RefIn:  false,
			RefOut: false,
			XorOut: 0x00,
			Check:  0xF7,
			Name:   "CRC-8/EGTS",
		})
    	crc := int(crc8.Checksum(response.Bytes(), table2))
    	response.WriteByte(byte(crc))
    
    	for i := 0; i < int(len(record.Bytes())); i++ {
        	response.WriteByte(record.Bytes()[i])
        }
    
    	binary.Write(response, binary.LittleEndian, uint16(recordChecksum))
    
    	_, err := conn.Write(response.Bytes())
		if err != nil {
    		res.error = err
		}
    	
	}
}

func (p *EgtsProtocol) handle(readbuff []byte, conn *net.TCPConn, imei string, bits Bitset) HandlerResponse {
	var records []Record
	params := make(map[string]interface{})

	bit := BitUtil{}
	
	res := HandlerResponse{}
	res.protocol = EGTS_PROTOCOL

	buff := bytes.NewBuffer(readbuff)
	buffer_length := len(readbuff)

	var headerLength uint8
	var index uint16
	var packetType uint8

	var gpstime uint32
	var lat uint32
	var lon uint32
	var speed uint16

	var b_course uint8
	var course uint16

	var alt int32
	var sat uint8

	var lat_float float64
	var lon_float float64
	useObjectIdAsDeviceId := false

	binary.Read(buff, binary.BigEndian, &headerLength)
	binary.Read(buff, binary.BigEndian, &index)
	binary.Read(buff, binary.BigEndian, &packetType)
	
	buff.Next(int(headerLength))

	if packetType == PT_RESPONSE {
    	return res
	}

	var objectId uint32
	for buff.Len() > 2 {
    	var length uint16
    	var recordIndex uint16
    	var recordFlags uint8
    
    	binary.Read(buff, binary.LittleEndian, &length)
		binary.Read(buff, binary.LittleEndian, &recordIndex)
		binary.Read(buff, binary.LittleEndian, &recordFlags)
    
    	if bit.check(recordFlags, 0) {
        	binary.Read(buff, binary.LittleEndian, &objectId)
    	}
    
    	if bit.check(recordFlags, 1) {
        	buff.Next(4)
    	}
    
    	if bit.check(recordFlags, 1) {
        	buff.Next(4)
    	}
    
    	var serviceType uint8
    	binary.Read(buff, binary.BigEndian, &serviceType)
    	buff.Next(1)
    
    	var recordEnd uint32
    	recordEnd = uint32(buffer_length - buff.Len()) + uint32(length);
    
    	response := bytes.NewBuffer(nil)
    	binary.Write(response, binary.LittleEndian, uint16(recordIndex))
    	response.WriteByte(0)
    
    	log.Println(length, recordIndex, recordEnd, recordFlags)
    
    	p.sendResponse(res, conn, PT_RESPONSE, int(index), int(serviceType), MSG_RECORD_RESPONSE, response)
    
    	for (buffer_length - buff.Len()) < int(recordEnd) {
        	var typee uint8
        	var end uint16
        
        	binary.Read(buff, binary.BigEndian, &typee)
        	binary.Read(buff, binary.LittleEndian, &end)
        	end = end + uint16(buffer_length - buff.Len())
        
        	if typee == MSG_TERM_IDENTITY {
            	useObjectIdAsDeviceId = false
            	
            	buff.Next(4) // object id
            	var flags uint8
            	binary.Read(buff, binary.BigEndian, &flags)
            
            	if bit.check(flags, 0) {
                	buff.Next(2) // home dispatcher identifier
                }
            
            	if bit.check(flags, 1) {
                	buff.Truncate(15)
        			imei := buff.String()
        			res.imei = imei
                }
            
            	if bit.check(flags, 2) {
                	buff.Truncate(16)
        			imei := buff.String()
        			res.imei = imei
                }
            
            	if bit.check(flags, 3) {
                	buff.Next(3) // language identifier
                }
            
            	if bit.check(flags, 5) {
                	buff.Next(3) // network identifier
                }
            
            	if bit.check(flags, 6) {
                	buff.Next(2) // buffer size
                }
            
            	if bit.check(flags, 7) {
                	buff.Truncate(15)
        			imei := buff.String()
        			res.imei = imei
                }
            
            	response := bytes.NewBuffer(nil)
            	response.WriteByte(0)
            	p.sendResponse(res, conn, PT_APPDATA, 0, int(serviceType), MSG_RESULT_CODE, response)
            } else if typee == MSG_POS_DATA {
            	binary.Read(buff, binary.LittleEndian, &gpstime)
            	gpstime = gpstime + 1262304000
            	binary.Read(buff, binary.LittleEndian, &lat)
            	binary.Read(buff, binary.LittleEndian, &lon)
            
            	lat_float = float64((lat * 90) / 0xfffffff)
            	lon_float = float64((lon * 180) / 0xfffffff)
            
            	var flags uint8
            	binary.Read(buff, binary.BigEndian, &flags)
            	
            	if bit.check(flags, 5) {
                	lat_float = -lat_float
                }
            
            	if bit.check(flags, 6) {
                	lon_float = -lon_float
            	}
            
            	binary.Read(buff, binary.LittleEndian, &speed)
            	speed = uint16(float32(bit.to(byte(speed), 14)) * 0.1)
            
            	binary.Read(buff, binary.BigEndian, &b_course)
            	course = uint16(b_course)
            	if bit.check(byte(speed), 15) {
                	course = course + 0x100
                }
            
            	params["odometer"] = readUnsignedMediumLE(buff.Bytes()[:3], 0) * 100
            	buff.Next(3)
            	var ign uint8
            	binary.Read(buff, binary.BigEndian, &ign)
            	params["ign"] = ign
            	var event uint8
            	binary.Read(buff, binary.BigEndian, &event)
            	params["event_id"] = event
            
            	if bit.check(flags, 7) {
                	alt = readMediumLE(buff.Bytes()[:3], 0)
               	 	buff.Next(3)
            	}
            
            } else if typee == MSG_EXT_POS_DATA {
            	var flags uint8
            	binary.Read(buff, binary.BigEndian, &flags)
            
            	if bit.check(flags, 0) {
                	var vdop uint16
                	binary.Read(buff, binary.LittleEndian, &vdop)
                	params["vdop"] = vdop
                }
            	
            	if bit.check(flags, 1) {
                	var hdop uint16
                	binary.Read(buff, binary.LittleEndian, &hdop)
                	params["hdop"] = hdop
                }
            
            	if bit.check(flags, 2) {
                	var pdop uint16
                	binary.Read(buff, binary.LittleEndian, &pdop)
                	params["pdop"] = pdop
                }
            
            	if bit.check(flags, 3) {
                	binary.Read(buff, binary.BigEndian, &sat)
                }
            	
            } else if typee == MSG_AD_SENSORS_DATA {
            	var inputMask uint8
            	binary.Read(buff, binary.BigEndian, &inputMask)
            	var key_output uint8
            	binary.Read(buff, binary.BigEndian, &key_output)
            	
            	params["key_output"] = key_output
            
            	var adcMask uint8
            	binary.Read(buff, binary.BigEndian, &adcMask)
            
            	for i := 0; i < 8; i++ {
                	if bit.check(inputMask, byte(i)) {
                    	var value uint8
                    	binary.Read(buff, binary.BigEndian, &value)
                    	params["in_" + strconv.Itoa(i + 1)] = value
                    }
                }
            
            	for i := 0; i < 8; i++ {
                	if bit.check(adcMask, byte(i)) {
                    	params["a_" + strconv.Itoa(i + 1)] = readUnsignedMediumLE(buff.Bytes()[:3], 0)
                   		buff.Next(3)
                    }
                }
            
            } else if typee == MSG_LIQUID_LEVEL_SENSOR {
            	var flags uint8
            	binary.Read(buff, binary.BigEndian, &flags)
            	buff.Next(2)
            
            	if bit.check(flags, 3) {
                	params["liquidRaw"] = 0
                } else {
                	var liquid uint32
                	binary.Read(buff, binary.LittleEndian, &liquid)
                	params["liquid"] = liquid
                }
            	
            }
        
        	if serviceType == SERVICE_TELEDATA {
            	if useObjectIdAsDeviceId && objectId != 0 {
                	// DevieSession
                	
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
            
            	// check DevieSession
        	}
    	}
    
    	res.records = records
    	
	}
	
	return res
}