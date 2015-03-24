package smtpd

import (
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/mail"
	"net/textproto"
	"strings"
	"sync"
)

type Message struct {
	From *mail.Address
	To   []*mail.Address

	Header mail.Header
	Body   []byte
}

func newMessage() *Message {
	return &Message{To: []*mail.Address{}}
}

func (m *Message) Read(r io.Reader) error {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return err
	}

	m.Header = msg.Header
	m.Body, err = ioutil.ReadAll(msg.Body)
	return err
}

var receiveChan chan mail.Message

type conn struct {
	rwc  net.Conn
	Text *textproto.Conn
	msg  *Message
	srv  *Server
	i    int
}

func (s *Server) newConn(rwc net.Conn) (c *conn, err error) {
	c = new(conn)
	c.msg = newMessage()
	c.srv = s
	c.rwc = rwc
	c.i = 0
	return c, nil
}

type stateFn func(c *conn) stateFn

func (c *conn) PrintfLine(format string, args ...interface{}) error {
	fmt.Printf(format, args...)
	fmt.Println("")
	return c.Text.PrintfLine(format, args...)
}

func (c *conn) ReadLine() (string, error) {
	s, err := c.Text.ReadLine()
	fmt.Println(s)
	return s, err
}

func startState(c *conn) stateFn {
	c.PrintfLine("220 Welcome to DutchCoders SMTPD (https://github.com/dutchcoders/smtpd).")
	return helloState
}

func unrecognizedState(c *conn) stateFn {
	c.PrintfLine("500 unrecognized command")
	return loopState
}

func errorState(format string, args ...interface{}) stateFn {
	msg := fmt.Sprintf(format, args...)
	return func(c *conn) stateFn {
		c.PrintfLine("500 %s", msg)
		return nil
	}
}

func isCommand(line string, cmd string) bool {
	return strings.HasPrefix(strings.ToUpper(line), cmd)
}

func mailFromState(c *conn) stateFn {
	line, _ := c.ReadLine()

	if isCommand(line, "RCPT TO") {
		addr, err := mail.ParseAddress(line[8:])
		if err != nil {
			return errorState("Could not parse address %s.", err)
		}

		c.msg.To = append(c.msg.To, addr)

		c.PrintfLine("250 Ok")
		return mailFromState
	} else if isCommand(line, "DATA") {
		c.PrintfLine("354 Enter message, ending with \".\" on a line by itself")

		defer func() {
			c.msg = newMessage()
		}()

		hasher := sha1.New()
		err := c.msg.Read(io.TeeReader(c.Text.DotReader(), hasher))
		if err == nil {
			if err = c.srv.handle(*c.msg); err != nil {
				return errorState(err.Error())
			}

			c.PrintfLine("250 OK : queued as +%x", hasher.Sum(nil))

		} else {
			return errorState("Error : error %s", err)
		}

		return loopState
	} else {
		return unrecognizedState
	}
}

func loopState(c *conn) stateFn {
	line, _ := c.ReadLine()

	c.i++

	if c.i > 100 {
		return errorState("Error: invalid.")
	}

	var err error

	if isCommand(line, "MAIL FROM") {
		c.msg.From, err = mail.ParseAddress(line[10:])
		if err != nil {
			return errorState("Could not parse address %s.", err)
		}

		c.PrintfLine("250 Ok")

		fmt.Println(c.msg.From)

		return mailFromState
	} else if isCommand(line, "STARTTLS") {
		c.PrintfLine("220 Ready to start TLS")

		tlsConn := tls.Server(c.rwc, &tls.Config{})

		if err := tlsConn.Handshake(); err != nil {
			fmt.Println(err)
			return nil
		}

		c.Text = textproto.NewConn(tlsConn)
		return helloState
	} else if isCommand(line, "RSET") {
		c.msg = newMessage()
		c.PrintfLine("250 Ok")
		return loopState
	} else if isCommand(line, "QUIT") {
		c.PrintfLine("221 Bye")
		return nil
	} else if isCommand(line, "NOOP") {
		c.PrintfLine("250 Ok")
		return loopState
	} else if strings.Trim(line, " \r\n") == "" {
		return loopState
	} else {
		return unrecognizedState
	}
}

func parseHelloArgument(arg string) (string, error) {
	domain := arg
	if idx := strings.IndexRune(arg, ' '); idx >= 0 {
		domain = arg[idx+1:]
	}
	if domain == "" {
		return "", fmt.Errorf("Invalid domain")
	}
	return domain, nil
}

func helloState(c *conn) stateFn {
	line, _ := c.ReadLine()

	if isCommand(line, "HELLO") {
		domain, err := parseHelloArgument(line)
		if err != nil {
			return errorState(err.Error())
		}

		c.PrintfLine("250 Hello %s", domain)
		return loopState
	} else if isCommand(line, "EHLO") {
		domain, err := parseHelloArgument(line)
		if err != nil {
			return errorState(err.Error())
		}

		c.PrintfLine("250 Hello %s", domain)
		c.PrintfLine("250-STARTTLS")
		c.PrintfLine("250-SIZE")
		c.PrintfLine("250-HELP")
		return loopState
	} else {
		return errorState("Before we shake hands it will be appropriate to tell me who you are.")
	}
}

type Handler interface {
	Handle(msg Message) error
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

func HandleFunc(handler func(msg Message) error) {
	DefaultServeMux.HandleFunc(handler)
}

var DefaultServeMux = NewServeMux()

func NewServeMux() *ServeMux { return &ServeMux{m: make([]HandlerFunc, 0)} }

type Server struct {
	Addr    string
	Handler Handler
}

func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		panic(err)
	}

	fmt.Println("SMTP Server listening.")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
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

func ListenAndServe(addr string) error {
	server := &Server{Addr: addr, Handler: DefaultServeMux}
	return server.ListenAndServe()
}

func (s *ServeMux) Handle(msg Message) error {
	for _, h := range s.m {
		if err := h(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) handle(msg Message) error {
	return s.Handler.Handle(msg)
}

func (c *conn) serve() {
	defer c.rwc.Close()

	c.Text = textproto.NewConn(c.rwc)

	for state := startState; state != nil; {
		state = state(c)
	}
}
