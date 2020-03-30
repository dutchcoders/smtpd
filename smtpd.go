package smtpd

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"os"
	"os/signal"
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
}

type smtpServer struct {
	Banner    func() string
	TLSConfig *tls.Config
	Handler   Handler

	starttls bool
}

func (s *Server) ListenAndServe() error {

	for _, ln := range s.Listeners {
		server := &smtpServer{
			Banner:    ln.Banner,
			TLSConfig: ln.TLSConfig,
			Handler:   ln.Handler,
		}

		switch ln.Mode {
		case "plain":
			//STARTTLS is optional.
			listen, err := net.Listen("tcp", ln.Address+":"+ln.Port)
			if err != nil {
				return fmt.Errorf("Listener: %+v, error: %w", ln, err)
			}

			server.starttls = true
			if server.TLSConfig == nil {
				server.starttls = false
			}

			go server.listenAndServe(listen)
		case "starttls":
			//TODO (jerry 2020-03-30): Force STARTTLS.
			if server.TLSConfig == nil {
				return fmt.Errorf("Mode: tls, need a tls.Config")
			}

			listen, err := net.Listen("tcp", ln.Address+":"+ln.Port)
			if err != nil {
				return fmt.Errorf("Listener: %+v, error: %w", ln, err)
			}

			server.starttls = true

			go server.listenAndServe(listen)
		case "tls":
			if server.TLSConfig == nil {
				return fmt.Errorf("Mode: tls, need a tls.Config")
			}

			listen, err := tls.Listen("tcp", ln.Address+":"+ln.Port, server.TLSConfig)
			if err != nil {
				return fmt.Errorf("Listener: %+v, error: %w", ln, err)
			}

			server.starttls = false

			go server.listenAndServe(listen)
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	<-sigs

	return nil
}

func (s *smtpServer) listenAndServe(ln net.Listener) {

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
}

func New(options ...func(*Config) error) (*Server, error) {
	cfg := &Config{
		Listeners: make([]Listener, 0, 2),
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

func (s *smtpServer) newConn(rwc net.Conn) (c *conn, err error) {
	c = &conn{
		server: s,
		rwc:    rwc,
		i:      0,
	}

	c.msg = c.newMessage()
	return c, nil
}

type serverHandler struct {
	srv *smtpServer
}

func (sh serverHandler) Serve(msg Message) {
	handler := sh.srv.Handler
	if handler == nil {
		handler = DefaultServeMux
	}

	handler.Serve(msg)
}
