package main

import (
	"bytes"
	"encoding/binary"
	//"log"
	"net"
	//"time"

	"strconv"
	"time"
	"errors"
	"math"
)

const (
	IMEI_PACK = 0x03
	PHOTO_PACK = 0x0A
	DATA_PACK = 0x01
	CMD_RESPONSE_SIZE = 0x84
	NEOMATICA_PROTOCOL = "Neomatica"
)

type NeomaticaProtocol struct {
}


func (p *NeomaticaProtocol) handle(readbuff []byte, conn *net.TCPConn, imei string, bits Bitset) HandlerResponse {
	res := HandlerResponse{}
	res.protocol = NEOMATICA_PROTOCOL

	buff := bytes.NewBuffer(readbuff)

	var device_id uint16
	var size byte
	var dtype byte
	var hw_type byte
	var reply_enabled byte
	var crc byte

	binary.Read(buff, binary.BigEndian, &device_id)
	binary.Read(buff, binary.BigEndian, &size)

	if size != CMD_RESPONSE_SIZE {
    	binary.Read(buff, binary.BigEndian, &dtype)
    	if dtype == IMEI_PACK {
        
        	imei, err := p.getIMEI(buff)
			if err != nil {
				res.error = err
			}
    		res.imei = imei
        
        	binary.Read(buff, binary.BigEndian, &hw_type)
    		binary.Read(buff, binary.BigEndian, &reply_enabled)
    		buff.Next(44)
    		binary.Read(buff, binary.BigEndian, &crc)
        } else {
        	res.imei = imei
        
        	records, err1 := p.getRecords(buff, dtype)
			if err1 != nil {
    			res.error = err1
			}
			res.records = records
        
        	_, err2 := conn.Write([]byte{0x01})
			if err2 != nil {
    			res.error = err2
			}
        }
    } else{
    	// TODO
    }

	return res
}

func (p *NeomaticaProtocol) getIMEI(buff *bytes.Buffer) (string, error) {
	var imei string

	buff.Truncate(15)

	imei = buff.String()

	if imei == "" {
		return "", errors.New("Imei is nil")
	}

	return padLeft(imei, "0", 15), nil
}

func (p *NeomaticaProtocol) getRecords(buff *bytes.Buffer, dtype byte) ([]Record, error) {
	var records []Record

	bit := BitUtil{}

	params := make(map[string]interface{})

	if bit.to(dtype, 2) == 0 {
    	var soft byte
   		var key_index uint16
    
    	var status uint16
    	var lat	float32
    	var lon	float32
    	
    	var course uint16
    	var speed uint16
    
    	var acc byte
    	var alt uint16
    	var hdop byte
    	var sat byte
   	 	var gpstime uint32
    	var vpower int16
    	var vbattery int16
    
    	binary.Read(buff, binary.LittleEndian, &soft)
    	binary.Read(buff, binary.LittleEndian, &key_index)
    	
    	binary.Read(buff, binary.LittleEndian, &status)
    	binary.Read(buff, binary.LittleEndian, &lat)
    	binary.Read(buff, binary.LittleEndian, &lon)
   
    	binary.Read(buff, binary.LittleEndian, &course)
    	binary.Read(buff, binary.LittleEndian, &speed)
    
    	binary.Read(buff, binary.LittleEndian, &acc)
    	binary.Read(buff, binary.LittleEndian, &alt)
    	binary.Read(buff, binary.LittleEndian, &hdop)
    	binary.Read(buff, binary.LittleEndian, &sat)
    	binary.Read(buff, binary.LittleEndian, &gpstime)
    	binary.Read(buff, binary.LittleEndian, &vpower)
    	binary.Read(buff, binary.LittleEndian, &vbattery)
    
    	params["hdop"] = math.Floor((float64(hdop) * 0.1)*10)/10
    	params["pwr_ext"] = math.Floor((float64(vpower) * 0.001)*100)/100
    	params["pwr_int"] = math.Floor((float64(vbattery) * 0.001)*100)/100
    	
    	if bit.check(dtype, 2) {
        	var vib byte
        	var vib_count byte
        	var out byte
        	var in_alarm byte
        
        	binary.Read(buff, binary.LittleEndian, &vib)
        	binary.Read(buff, binary.LittleEndian, &vib_count)
        	binary.Read(buff, binary.LittleEndian, &out)
        	binary.Read(buff, binary.LittleEndian, &in_alarm)
        
        	var i byte
     
        	for ; i <= 3; i++ {
            	if bit.check(out, i) {
                	params["out_" + strconv.Itoa(int(i)+1)] = 1
                } else {
                	params["out_" + strconv.Itoa(int(i)+1)] = 0
                }
            }
        
        	params["vib"] = vib
        	params["vib_count"] = vib_count
        	params["in_alarm"] = in_alarm
    	}
    
    	if bit.check(dtype, 3) {
        	for i := 1; i <= 6; i++ {
            	var ina uint16
            	binary.Read(buff, binary.LittleEndian, &ina)
            	params["in_a" + strconv.Itoa(i)] = ina
        	}
    	}
    
    	if bit.check(dtype, 4) {
        	for i := 1; i <= 2; i++ {
            	var ind uint32
            	binary.Read(buff, binary.LittleEndian, &ind)
            	params["in_d" + strconv.Itoa(i)] = ind
            }
    	}
    
    	if bit.check(dtype, 4) {
        	for i := 1; i <= 3; i++ {
            	var fuel uint16
            	binary.Read(buff, binary.LittleEndian, &fuel)
            	params["fuel" + strconv.Itoa(i)] = fuel
            }
        
        	for i := 1; i <= 3; i++ {
            	var temp uint8
            	binary.Read(buff, binary.LittleEndian, &temp)
            	params["temp" + strconv.Itoa(i)] = temp
            }
        }
    
    	if bit.check(dtype, 6) {
        	// TODO        	
    	}
    
    	if bit.check(dtype, 7) {
        	var odm uint32
        	binary.Read(buff, binary.LittleEndian, &odm)
        	params["odometr"] = odm
    	}
    

    	record := Record{
        	int(gpstime),
        	int(time.Now().Unix()),
        	Pos{
            	math.Floor(float64(lat) * 1000000) / 1000000,
            	math.Floor(float64(lon) * 1000000) / 1000000,
            	int(alt),
            	int(float32(course) * 0.1),
            	int(speed),
            	int(sat & 0x0f),
        	},
        	params,
        }
    	records = append(records, record)
    	return records, nil
	}
	return nil, errors.New("No record is type")
}