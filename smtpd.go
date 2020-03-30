package smtpd

import (
	"net"
	"net/mail"
	"sync"

	logging "github.com/op/go-logging"
)

var log = logging.MustGetLogger("smtpd")

var receiveChan chan mail.Message

type Handler interface {
	Serve(msg Message) error
}

type HandlerFunc func(msg Message) error

type ServeMux struct {
	m  []HandlerFunc
	mu sync.RWMutex
}

func (mux *ServeMux) HandleFunc(handler func(msg Message) error) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	mux.m = append(mux.m, handler)
}

func HandleFunc(handler func(msg Message) error) *ServeMux {
	DefaultServeMux.HandleFunc(handler)
	return DefaultServeMux
}

var DefaultServeMux = NewServeMux()

func NewServeMux() *ServeMux { return &ServeMux{m: make([]HandlerFunc, 0)} }

type Server struct {
	*Config
	Handler Handler
}

func (s *Server) ListenAndServe(handler Handler) error {
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Error("Error accept: %s", err.Error())
			continue
		}

		c, err := s.newConn(conn)
		if err != nil {
			continue
		}

		go c.serve()
	}

	return nil
}

func New(options ...func(*Config) error) (*Server, error) {
	cfg := &Config{
		Banner: func() string {
			return "DutchCoders SMTPd"
		},
		TLSConfig: nil,
	}

	for _, optionFn := range options {
		if err := optionFn(cfg); err != nil {
			return nil, err
		}
	}

	server := &Server{
		Config: cfg,
	}

	return server, nil
}

func (s *ServeMux) Serve(msg Message) error {
	for _, h := range s.m {
		if err := h(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) newConn(rwc net.Conn) (c *conn, err error) {
	c = &conn{
		server: s,
		rwc:    rwc,
		i:      0,
	}

	c.msg = c.newMessage()
	return c, nil
}

type serverHandler struct {
	srv *Server
}

func (sh serverHandler) Serve(msg Message) {
	handler := sh.srv.Handler
	if handler == nil {
		handler = DefaultServeMux
	}

	handler.Serve(msg)
}
