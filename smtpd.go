package smtpd

import (
	"context"
	"crypto/tls"
	"errors"
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
}

type smtpServer struct {
	ID        string
	Banner    func() string
	TLSConfig *tls.Config
	Handler   Handler

	starttls bool
}

var ErrServerClosed = errors.New("SMTPd Closed.")

//ListenAndServe starts serving smtp on the configured listeners.
//Always returns an error.
func (s *Server) ListenAndServe(ctx context.Context) error {
	//lctx, cancel := context.WithCancel(ctx)
	//defer cancel()

	wg := &sync.WaitGroup{}

	log.Debugf("Starting %d listeners.", len(s.Listeners))

	//keep track of listeners to close them.
	listeners := make([]*onceCloseListener, 0, 2)

	for _, ln := range s.Listeners {
		server := &smtpServer{
			ID:        ln.ID,
			Banner:    ln.Banner,
			TLSConfig: ln.TLSConfig,
			Handler:   ln.Handler,
		}

		var (
			listen net.Listener
			err    error
		)

		switch ln.Mode {
		case "plain":
			//STARTTLS is optional.
			listen, err = net.Listen("tcp", ln.Address+":"+ln.Port)
			if err != nil {
				return fmt.Errorf("Listener: %+v, error: %w", ln, err)
			}

			server.starttls = true
			if server.TLSConfig == nil {
				server.starttls = false
			}
		case "starttls":
			//TODO (jerry 2020-03-30): Force STARTTLS.
			if server.TLSConfig == nil {
				return fmt.Errorf("Mode: tls, need a tls.Config")
			}

			listen, err = net.Listen("tcp", ln.Address+":"+ln.Port)
			if err != nil {
				return fmt.Errorf("Listener: %+v, error: %w", ln, err)
			}

			server.starttls = true
		case "tls":
			if server.TLSConfig == nil {
				return fmt.Errorf("Mode: tls, need a tls.Config")
			}

			listen, err = tls.Listen("tcp", ln.Address+":"+ln.Port, server.TLSConfig)
			if err != nil {
				return fmt.Errorf("Listener: %+v, error: %w", ln, err)
			}

			server.starttls = false
		}

		l := &onceCloseListener{Listener: listen}

		wg.Add(1)
		go server.serve(l, wg)
		listeners = append(listeners, l)

		log.Infof("Server: '%s' start serving on port %s in %s mode.", ln.ID, ln.Port, ln.Mode)
	}

	if len(listeners) == 0 {
		return fmt.Errorf("No Listeners started; %w", ErrServerClosed)
	}

	//wait for cancelation for a clean shutdown.
	<-ctx.Done()

	log.Info("SMTPd shutting down...")

	for _, l := range listeners {
		l.Close()
	}
	wg.Wait()

	return ErrServerClosed
}

func (s *smtpServer) serve(ln net.Listener, wg *sync.WaitGroup) {
	defer ln.Close()
	defer wg.Done()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				log.Debugf("Accept error: %v; retrying", err)
				continue
			}
			log.Errorf("Error accept (Listener: %s): %v; closing", s.ID, err)
			return
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

//onceCloseListener wraps a net.Listener. Protecting it from multiple Close calls.
type onceCloseListener struct {
	net.Listener
	once     sync.Once
	closeErr error
}

func (oc *onceCloseListener) Close() error {
	oc.once.Do(oc.close)
	return oc.closeErr
}

func (oc *onceCloseListener) close() {
	oc.closeErr = oc.Listener.Close()
}
