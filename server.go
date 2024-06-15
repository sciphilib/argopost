package smtp

import (
	"fmt"
	"log"
	"net"
)

type Server struct {
	host string
	port string
	sm   *SessionManager
}

type Config struct {
	Host string
	Port string
}

func New(config *Config) *Server {
	return &Server{
		host: config.Host,
		port: config.Port,
		sm:   NewManager(),
	}
}

func (server *Server) Run() {
	host := fmt.Sprintf("%s:%s", server.host, server.port)

	listener, err := net.Listen("tcp", host)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		session := server.sm.CreateSession(conn)
		go server.handle(session)
	}
}

func (server *Server) handle(s *Session) {
	defer server.sm.CloseSession(s)
	err := server.sm.HandleSession(s)
	if err != nil {
		log.Fatal(err)
	}
}
