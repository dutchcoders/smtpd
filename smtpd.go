package smtpd

import (
	"context"
	"crypto/tls"
	"fmt"
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

func (s *Server) ListenAndServe(ctx context.Context, handler Handler) error {
	lctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, ln := range s.Listeners {
		switch ln.Mode {
		case "plain":
			//STARTTLS is optional.
			listen, err := net.Listen("tcp", ln.Address+":"+ln.Port)
			if err != nil {
				return fmt.Errorf("Listener: %+v, error: %w", ln, err)
			}
			go s.listenAndServe(lctx, listen, true)
		case "starttls":
			//TODO (jerry 2020-03-30): Force STARTTLS.
			listen, err := net.Listen("tcp", ln.Address+":"+ln.Port)
			if err != nil {
				return fmt.Errorf("Listener: %+v, error: %w", ln, err)
			}
			go s.listenAndServe(lctx, listen, true)
		case "tls":
			//STARTTLS needs to be disabled.
			listen, err := tls.Listen("tcp", ln.Address+":"+ln.Port, s.TLSConfig)
			if err != nil {
				return fmt.Errorf("Listener: %+v, error: %w", ln, err)
			}
			go s.listenAndServe(lctx, listen, false)
		}
	}

	<-lctx.Done()

	return nil
}

func (s *Server) listenAndServe(ctx context.Context, ln net.Listener, starttls bool) {
	wg := sync.WaitGroup{}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Error("Error accept: %s", err.Error())
			continue
		}

		c, err := s.newConn(conn, starttls)
		if err != nil {
			continue
		}

		wg.Add(1)
		go c.serve()

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
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

func (s *Server) newConn(rwc net.Conn, starttls bool) (c *conn, err error) {
	c = &conn{
		server:   s,
		rwc:      rwc,
		i:        0,
		starttls: starttls,
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
