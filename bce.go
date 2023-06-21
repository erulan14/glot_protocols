package main

import (
	"log"
	"encoding/binary"
	"bytes"
	"net"
	"strconv"
	"time"
	"fmt"

	"container/list"
)

type BceProtocol struct {
}

const (
	BCE_PROTOCOL = "BCE"
	DATA_TYPE = 7
	MSG_ASYNC_STACK = 0xA5
	MSG_STACK_COFIRM = 0x19
	MSG_TIME_TRIGGERED = 0xA0
	MSG_OUTPUT_CONTROL = 0x41
	MSG_OUTPUT_CONTROL_ACK = 0xc1
)

func (p *BceProtocol) decodeMask1(buff *bytes.Buffer, mask byte, params map[string]interface{}) {
	bit := BitUtil{}

	if bit.check(mask, 1) {
    	var value uint16
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["input"] = value
	}

	for i := 1; i <= 8; i++ {
    	if bit.check(mask, byte(i + 1)) {
        	var value uint16
        	binary.Read(buff, binary.LittleEndian, &value)
        	params["prefix_adc" + strconv.Itoa(i)] = value
        }
    }

	if bit.check(mask, 10) {
    	buff.Next(4)
    }

	if bit.check(mask, 11) {
    	buff.Next(4)
    }

	if bit.check(mask, 12) {
    	var value uint16
    	binary.Read(buff, binary.BigEndian, &value)
    	params["fuel1"] = value
    }

	if bit.check(mask, 13) {
    	var value uint16
    	binary.Read(buff, binary.BigEndian, &value)
    	params["fuel2"] = value
    }

	if bit.check(mask, 14) {
    	buff.Next(9) // Network cell
    }
}


func (p *BceProtocol) decodeMask2(buff *bytes.Buffer, mask byte, params map[string]interface{}) {
	bit := BitUtil{}

	if bit.check(mask, 0) {
    	buff.Next(2) // wheel speed
	}
	if bit.check(mask, 1) {
    	buff.Next(1) // acceleration pedal
	}
	if bit.check(mask, 2) {
    	var value uint32
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["fuel_used"] = uint32(float32(value) * 0.5)
	}
	if bit.check(mask, 3) {
    	var value uint8
    	binary.Read(buff, binary.BigEndian, &value)
    	params["fuel_level"] = value
	}
	if bit.check(mask, 4) {
    	var value uint16
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["rpm"] = int(float32(value) * 0.125)
	}
	if bit.check(mask, 5) {
    	var value uint32
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["hours"] = value
	}
	if bit.check(mask, 6) {
    	var value uint32
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["odometer"] = value
	}
	if bit.check(mask, 7) {
    	var value byte
    	binary.Read(buff, binary.BigEndian, &value)
    	params["coolantTemp"] = value - 40
	}
	if bit.check(mask, 8) {
    	var value uint8
    	binary.Read(buff, binary.BigEndian, &value)
    	params["fuel2"] = value
	}
	if bit.check(mask, 9) {
    	var value uint8
    	binary.Read(buff, binary.BigEndian, &value)
    	params["engine_load"] = value
	}
	if bit.check(mask, 10) {
    	var value uint16
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["service_odometer"] = value
	}
	if bit.check(mask, 11) {
    	buff.Next(8) // sensors
	}
	if bit.check(mask, 12) {
    	buff.Next(2) // ambient air temperature
	}
	if bit.check(mask, 13) {
    	buff.Next(8) // trailer id
	}
	if bit.check(mask, 14) {
    	var value uint16
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["fuel_consumption"] = value
	}
}

func (p *BceProtocol) decodeMask3(buff *bytes.Buffer, mask byte, params map[string]interface{}) {
	bit := BitUtil{}

	if bit.check(mask, 0) {
    	buff.Next(2) // wheel speed
	}
	if bit.check(mask, 1) {
    	var value uint32
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["fuel_consumption"] = value
	}
	if bit.check(mask, 2) {
    	params["axle_weight"] = readMediumLE(buff.Bytes()[:3], 0)
        buff.Next(3)
	}
	if bit.check(mask, 3) {
    	buff.Next(1) // mil status
	}
	if bit.check(mask, 4) {
    	buff.Next(20) // dtc
	}
	if bit.check(mask, 5) {
    	buff.Next(2)
	}
	if bit.check(mask, 6) {
    	var value int64
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["driver_unique_id"] = value
	}
	if bit.check(mask, 7) {
    	var value uint16
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["prefix_temp1"] = uint32(float32(value) * 0.1 - 273)
	}
	if bit.check(mask, 8) {
    	buff.Next(2)
	}
	if bit.check(mask, 9) {
    	var value uint16
    	var value2 byte
    
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["fuel1"] = value
    
    	binary.Read(buff, binary.BigEndian, &value2)
   		params["fuelTemp1"] = value2
    
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["fuel2"] = value
    
    	binary.Read(buff, binary.BigEndian, &value2)
    	params["fuelTemp2"] = value2
    	
	}
	if bit.check(mask, 10) {
    	var value uint16
    	var value2 byte
    
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["fuel3"] = value
    
    	binary.Read(buff, binary.BigEndian, &value2)
   		params["fuelTemp3"] = value2
    
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["fuel4"] = value
    
    	binary.Read(buff, binary.BigEndian, &value2)
    	params["fuelTemp4"] = value2
	}
	if bit.check(mask, 11) {
    	buff.Next(21) // j1979 group 1
	}
	if bit.check(mask, 12) {
    	buff.Next(20) // f1979 dtc
	}
	if bit.check(mask, 13) {
    	buff.Next(9) // 1708 group 1
	}
	if bit.check(mask, 14) {
    	buff.Next(21) // driving quality
	}
}

func (p *BceProtocol) decodeMask4(buff *bytes.Buffer, mask byte, params map[string]interface{}) {
	bit := BitUtil{}

	if bit.check(mask, 0) {
    	buff.Next(4)
	}
	if bit.check(mask, 1) {
    	buff.Next(30) // lls group 3
	}
	if bit.check(mask, 2) {
    	buff.Next(4) // instant fuel consumption
	}
	if bit.check(mask, 3) {
    	buff.Next(10) // axle weight group
	}
	if bit.check(mask, 4) {
    	buff.Next(1)
	}
	if bit.check(mask, 5) {
    	buff.Next(2)
	}
	if bit.check(mask, 6) {
    	var value uint8
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["maxAcceleration"] = uint16(float32(value) * 0.02)
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["maxBraking"] = uint16(float32(value) * 0.02)
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["maxCornering"] = uint16(float32(value) * 0.02)
	}
	if bit.check(mask, 7) {
    	buff.Next(16)
	}
	if bit.check(mask, 8) {
    	for i := 0; i <= 4; i++ {
        	var temperature uint16
        	binary.Read(buff, binary.LittleEndian, &temperature)
        	if (temperature > 0) {
            	params["prefix_temp" + strconv.Itoa(i)] = uint16(float32(temperature) * 0.1)
            }
        	buff.Next(8)
        }
	}
	if bit.check(mask, 9) {
    	params["driver1"] = string(buff.Next(16))
    	params["driver2"] = string(buff.Next(16))
	}
	if bit.check(mask, 10) {
    	var value uint16
    	binary.Read(buff, binary.LittleEndian, &value)
    	params["odometer"] = value
	}
}



func (p *BceProtocol) handle(readbuff []byte, conn *net.TCPConn, imei string, bits Bitset) HandlerResponse {
	bit := BitUtil{}
	
	var records []Record
	params := make(map[string]interface{})

	res := HandlerResponse{}
	res.protocol = BCE_PROTOCOL

	buff := bytes.NewBuffer(readbuff)

	var value int64
	binary.Read(buff, binary.LittleEndian, &value)
	imei = fmt.Sprintf("%015d", value)
	res.imei = imei

	valid := false

	var gpstime uint32
	var lon float32
	var lat float32
	var alt uint16
	var course uint8
	var sat uint8
	var speed uint8

	for (buff.Len() > 1) {
    	var value uint16
    	binary.Read(buff, binary.LittleEndian, &value)
    	dataEnd := int(value) + (buff.Cap() - buff.Len())
    
    	var typec uint8
    	binary.Read(buff, binary.LittleEndian, &typec)
    
    	if typec != MSG_ASYNC_STACK && typec != MSG_TIME_TRIGGERED {
        	break
    	}
    
    	var confirmValue uint8
    	binary.Read(buff, binary.LittleEndian, &confirmValue)
    	confirmKey := confirmValue & 0x7F
    
    	for (buff.Cap() - buff.Len() < dataEnd) {
        	var value uint8
    		binary.Read(buff, binary.LittleEndian, &value)
        	structEnd := int(value) + (buff.Cap() - buff.Len())
        
        	//var time uint32
    		binary.Read(buff, binary.LittleEndian, &gpstime)
        
        	if (gpstime & 0x0f) == DATA_TYPE {
            	gpstime = gpstime >> 4 << 1
            	gpstime += 0x47798280 // 01/01/2008
            
            	var mask byte
            	masks := list.New()
            	for {
					binary.Read(buff, binary.LittleEndian, &mask)
                	masks.PushBack(mask)
                	if bit.check(byte(mask), 15) {
						break
					}
				}
            
            	mask1 := masks.Front()
            	mask = mask1.Value.(byte)
            	
            	if bit.check(mask, 0) {
                	valid = true
                	binary.Read(buff, binary.LittleEndian, &lon)
					binary.Read(buff, binary.LittleEndian, &lat)
                	binary.Read(buff, binary.BigEndian, &speed)
                	var status uint8
                	binary.Read(buff, binary.LittleEndian, &status)
                	sat = bit.to(status, 4)
                	params["hdop"] = bit.from(status, 4)
                
                	var courseValue uint8
                	binary.Read(buff, binary.BigEndian, &courseValue)
                	course = courseValue * 2
                
                	var altValue uint16
                	binary.Read(buff, binary.LittleEndian, &altValue)
                	alt = altValue
                
                	var odometerValue uint32
                	binary.Read(buff, binary.LittleEndian, &odometerValue)
                	params["odometer"] = odometerValue
            	}
            
            	p.decodeMask1(buff, mask, params)
            	
            	if masks.Len() >= 2 {
                	p.decodeMask2(buff, mask1.Next().Value.(byte), params)
            	}
            
            	if masks.Len() >= 3 {
                	p.decodeMask3(buff, mask1.Next().Value.(byte), params)
            	}
            
            	if masks.Len() >= 4 {
                	p.decodeMask4(buff, mask1.Next().Value.(byte), params)
            	}
        	}
        
        	buff.Next(structEnd)
        
        	if valid {
            	record := Record{
            		int(gpstime),
        			int(time.Now().Unix()),
            		Pos{
                    	float64(lat),
                    	float64(lon),
                    	int(alt),
            			int(course),
            			int(speed),
            			int(sat),
            		},
            		params,
            	}
        
        		records = append(records, record)
            	log.Println(records)
        	}
    	}
    
    	if typec == MSG_ASYNC_STACK {
        	response := bytes.NewBuffer(nil)
        	num, _ := strconv.ParseInt(imei, 10, 64)
        	binary.Write(response, binary.LittleEndian, num)
        	binary.Write(response, binary.LittleEndian, uint16(2))
        	response.WriteByte(MSG_STACK_COFIRM)
        	response.WriteByte(confirmKey)
        
        	var checksum byte
        	for i := 1; i <= response.Len(); i++ {
            	checksum += response.Bytes()[i]
            }
        	response.WriteByte(checksum);
        
        	_, err := conn.Write(response.Bytes())
			if err != nil {
    			res.error = err
			}
    	}
	}
	
	return res
}