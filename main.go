package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	colorReset = "\033[0m"
    colorRed = "\033[31m"
    colorGreen = "\033[32m"
    colorYellow = "\033[33m"
    colorBlue = "\033[34m"
    colorPurple = "\033[35m"
    colorCyan = "\033[36m"
    colorWhite = "\033[37m"
)

var (
	INFO    = string(colorGreen) + "[INFO]" + string(colorReset)
	WARNING = string(colorYellow)+ "[WARNING]" + string(colorReset)
	ERROR   = string(colorRed) +   "[ERROR]" + string(colorReset)
	CONNECT = string(colorBlue) +  "[CONNECT]" + string(colorReset)
	DISCONNECT = string(colorBlue)+"[DISCONNECT]" + string(colorReset)
)

type Mongodb struct {
	Host string `json:"host"`
	User string `json:"user"`
	Pass string `json:"pass"`
	Name string `json:"name"`
	Col string  `json:"col"`
}

type Redis struct {
	Host string `json:"host"`
	Db int `json:"db"`
}

type Postgresql struct {
	Host string `json:"host"`
	User string `json:"user"`
	Pass string `json:"pass"`
	Name string `json:"name"`
}

type DbConfigs struct {
	MongoDB Mongodb
	Redis Redis 
	PostgreSQL Postgresql
}

type Config struct {
	Host      string                 `json:"host"`
	Db        *DbConfigs			 `json:"db"`	//map[string]interface{} `json:"db"`
	Protocols []Protocol             `json:"protocols"`
}

func main() {
	log.Println(INFO, "Initializing packages, loading protocols and config")

	file, err := os.Open("config.json")
	if err != nil {
		log.Fatalln(ERROR, err)
		return
	}

	config := Config{}
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		log.Fatalln(ERROR, err)
		return
	}
	count_protocols := len(config.Protocols)
	log.Println(INFO, "Initialization completed successfully")
	log.Println(INFO, "Total number of protocols", count_protocols)

	var servers []*Server

	host := config.Host

	for i := 0; i < count_protocols; i++ {
		protocol_name := config.Protocols[i].Name
		protocol_port := config.Protocols[i].Port

		if config.Protocols[i].Enabled {
			var protocol_handler ProtocolHandler

			log.Println(INFO ,"Loading protocol", protocol_name)

			switch protocol_name {
			case "Teltonika":
				protocol_handler = ProtocolHandler(&TeltonikaProtocol{})
			case "Ruptela":
				protocol_handler = ProtocolHandler(&RuptelaProtocol{})
            case "Neomatica":
            	protocol_handler = ProtocolHandler(&NeomaticaProtocol{})
            case "Navtelecom":
            	protocol_handler = ProtocolHandler(&NavtelecomProtocol{})
            case "GalileoSky":
            	protocol_handler = ProtocolHandler(&GalileoskyProtocol{})
            case "Arnavi":
            	protocol_handler = ProtocolHandler(&ArnaviProtocol{})
            case "EGTS":
            	protocol_handler = ProtocolHandler(&EgtsProtocol{})
            case "BCE":
            	protocol_handler = ProtocolHandler(&BceProtocol{})
			default:
				log.Println(WARNING, "loading protocol", protocol_name)
				continue
			}

			s := NewServer(protocol_name, config.Db, protocol_handler)

			s.Start(host, protocol_port)

			servers = append(servers, s)
		}
	}

	log.Println(INFO, "All servers are up and running")

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	log.Println(INFO, "Signal received", <-ch)

	stopServers(servers)

	log.Println(INFO, "The program is stopped")
}