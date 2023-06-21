package main

import (
	"log"
	"net"
	"sync"
	"time"
	"github.com/redis/go-redis/v9"
	"encoding/json"
	"context"
	"errors"
	"strconv"
	"os"
	"bytes"

	//"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProtocolHandler interface {
	handle([]byte, *net.TCPConn, string, Bitset) HandlerResponse
}

type HandlerResponse struct {
	error   error
	imei    string
	protocol string
	records []Record
	bits    Bitset
}

type Protocol struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Port    string `json:"port"`
	Enabled bool   `json:"enabled"`
}

type Server struct {
	name      string
	ch        chan bool
	waitGroup *sync.WaitGroup
	dbConfigs  *DbConfigs
	redis 	  *redis.Client
	pgsql 	  *pgxpool.Pool
	mongo *mongo.Client
	listener  *net.TCPListener
	protocol  ProtocolHandler
	f *os.File
}

type Pos struct {
	X  float64 `json:"x" bson:"x"`
	Y  float64 `json:"y" bson:"y"`
	Z  int 	   `json:"z" bson:"z"`
	A  int 	   `json:"a" bson"a"`
	S  int     `json:"s" bson:"s"`
	St int     `json:"st" bson:"st"`
}

type Record struct {
	T        int    `json:"t" bson:"t"`
	ST		 int 	`json:"st" bson:"st"`
	Pos      Pos    `json:"pos" bson:"pos"`
	Params 	 map[string]interface{} `json:"p" bson:"p"`
}

var ctx = context.Background()

func NewServer(name string, dbConfigs *DbConfigs, protocol ProtocolHandler) *Server {
	log.Println(INFO, "Creating a server", name)

	url := "postgres://"+dbConfigs.PostgreSQL.User+":"+dbConfigs.PostgreSQL.Pass+"@"+dbConfigs.PostgreSQL.Host+"/"+dbConfigs.PostgreSQL.Name
	conn, err := pgxpool.Connect(ctx, url)
	if err != nil {
		log.Fatal(ERROR, "Unable to connect to database: %v\n", err)
	}

	uri := "mongodb://"+dbConfigs.MongoDB.User+":"+dbConfigs.MongoDB.Pass+"@"+dbConfigs.MongoDB.Host+"/"
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
    	log.Fatal(ERROR, err)
	}

	var redisClient = redis.NewClient(&redis.Options{
    	Addr: dbConfigs.Redis.Host,
    	DB: dbConfigs.Redis.Db,
	})

	f, err := os.OpenFile("/var/log/go_errors.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
    	log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)

	s := Server{
		name:      name,
		ch:        make(chan bool),
		waitGroup: &sync.WaitGroup{},
    	dbConfigs: dbConfigs,
    	redis: redisClient,
    	pgsql: conn,
    	mongo: client,
		protocol:  protocol,
    	f: f,
	}

	s.waitGroup.Add(1)
	return &s
}

func (s *Server) Start(host string, port string) {
	log.Println(INFO, "Running the server:", s.name)
	s.Listen(host, port)
	go s.Serve()
}

func (s *Server) Listen(host string, port string) {

	laddr, _ := net.ResolveTCPAddr("tcp", host+":"+port)

	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		log.Fatalln(WARNING, "Program can't open listening socket:", err.Error())
	}
	// defer listener.Close()

	log.Println(INFO, "Socket successfully run on the address:", listener.Addr())

	s.listener = listener
}

func (s *Server) Serve() {
	defer s.waitGroup.Done()
	for {
		select {
		case <-s.ch:
			log.Println(INFO, "Stopping an address at", s.listener.Addr())
			s.listener.Close()
			return
		default:
		}
		s.listener.SetDeadline(time.Now().Add(5e9)) // 5 секунд
		conn, err := s.listener.AcceptTCP()
		if nil != err {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			log.Println(WARNING, err)
		}
		//log.Println(CONNECT, conn.RemoteAddr())
		// conn.SetKeepAlive(true)
		s.waitGroup.Add(1)
		go s.HandleRequest(conn)
	}
}

func (s *Server) HandleRequest(conn *net.TCPConn) {
	//defer log.Println(DISCONNECT, conn.RemoteAddr())
	defer conn.Close()
	defer s.waitGroup.Done()

	var imei string
	var bits Bitset

	var i int = 0

	for {
		select {
		case <-s.ch:
			return
		default:
		}

		//conn.SetReadDeadline(time.Now().Add(5e9)) // 5 секунд

		readbuff := make([]byte, 2048) // TODO: Что делать, если ввод больше 2048?
		var buf bytes.Buffer
    	
    	for {
        	select {
			case <-s.ch:
				return
			default:
			}
        
        	conn.SetReadDeadline(time.Now().Add(5e9))
        
			n, err := conn.Read(readbuff)
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					// После 60 таймаутов (5 минут) закрываем соединение
					if i >= 60 {
						return
					} else {
						i++
						continue
					}
				}
				log.Println(WARNING, "Connection Read:", err)
				return
			} else {
				i = 0
			}
        
        	buf.Write(readbuff[:n])
        	if n < len(readbuff) {
        		break
            }
        	readbuff = make([]byte, len(readbuff)*2)
        }
        	

    	res := s.protocol.handle(buf.Bytes(), conn, imei, bits)
		if res.error != nil {
        	log.Println(WARNING, res.error)
        	//log.Println(res.records)
        	if res.error == errors.New("crc16") {
            	return
            }
		}
    
		imei = res.imei
    	bits = res.bits
    
    	var id int
    	err := s.pgsql.QueryRow(ctx, "SELECT id FROM main_unit WHERE uid=$1",imei).Scan(&id)
		if err != nil {
        	log.Println(WARNING, "POSTGRESQL:", imei, err)
			return
		}

    	if res.error != errors.New("crc16") {
        	if len(res.records) > 0 {
				s.SaveRecords(res, id)
        		//log.Println(INFO, res)
			}
   		}
	}
}

func (s *Server) SaveRecords(res HandlerResponse, id int) {
	for _, record := range res.records {
    	payload, err := json.Marshal(record)
    
        if err != nil {
            log.Println(WARNING, err)
        }
    
   	 	coll := s.mongo.Database(s.dbConfigs.MongoDB.Name).Collection(strconv.Itoa(id))
    
    	_, err = coll.InsertOne(context.TODO(), record)
    
    	if err != nil {
        	//log.Println(WARNING, err)
    	}
    
    	var f map[string]interface{}
		err = json.Unmarshal(payload, &f)
    
    	if err != nil {
        	log.Println(WARNING, err)
    	}
    
    	f["id"] = id
    	payload, err = json.Marshal(f)
    
        if err != nil {
            log.Println(WARNING, err)
        }
    
    	now := time.Now().Unix()
    
    	if int(now) - record.T < 600 {
        	if err:= s.redis.Publish(ctx, "realtime", payload).Err(); err != nil {
        		log.Println(WARNING, err)
        	}
    	}

	}
}

func (s *Server) Stop() {
	log.Println(INFO, "Server shutdown:", s.name)
	// s.mongoSession.Close()
	// s.listener.Close()
	s.f.Close()
	s.redis.Close()
	s.pgsql.Close()
	s.mongo.Disconnect(ctx)
	close(s.ch)
	s.waitGroup.Wait()
}

func stopServers(servers []*Server) {

	for _, server := range servers {
		server.Stop()
	}
}
