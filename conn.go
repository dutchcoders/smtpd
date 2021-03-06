package smtpd

import (
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/textproto"
	"strconv"
	"strings"
)

type conn struct {
	rwc    net.Conn
	Text   *textproto.Conn
	domain string
	msg    *Message
	server *smtpServer
	i      int
}

func (c *conn) RemoteAddr() net.Addr {
	return c.rwc.RemoteAddr()
}

type stateFn func(c *conn) stateFn

func (c *conn) PrintfLine(format string, args ...interface{}) error {
	fmt.Printf("< ")
	fmt.Printf(format, args...)
	fmt.Println("")
	return c.Text.PrintfLine(format, args...)
}

func (c *conn) ReadLine() (string, error) {
	s, err := c.Text.ReadLine()
	fmt.Printf("> ")
	fmt.Println(s)
	return s, err
}

func startState(c *conn) stateFn {
	c.PrintfLine("220 %s", c.server.Banner())
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

func outOfSequenceState() stateFn {
	return func(c *conn) stateFn {
		c.PrintfLine("503 command out of sequence")
		return nil
	}
}

func isCommand(line string, cmd string) bool {
	return strings.HasPrefix(strings.ToUpper(line), cmd)
}

func mailFromState(c *conn) stateFn {
	line, err := c.ReadLine()
	if err != nil {
		log.Error("[mailFromState] error: %s", err.Error())
		return nil
	}

	if line == "" {
		return loopState
	}

	if isCommand(line, "RSET") {
		c.PrintfLine("250 Ok")

		c.msg = c.newMessage()
		return loopState
	} else if isCommand(line, "RCPT TO") {
		addr, _ := mail.ParseAddress(line[8:])

		c.msg.To = append(c.msg.To, addr)

		c.PrintfLine("250 Ok")
		return mailFromState
	} else if isCommand(line, "BDAT") {
		parts := strings.Split(line, " ")

		count := int64(0)
		var err error
		if count, err = strconv.ParseInt(parts[1], 10, 32); err != nil {
			return errorState("[bdat]: error %s", err)
		}

		if _, err = io.CopyN(c.msg.Buffer, c.Text.R, count); err != nil {
			return errorState("[bdat]: error %s", err)
		}

		last := (len(parts) == 3 && parts[2] == "LAST")
		if !last {
			c.PrintfLine("250 Ok")
			return mailFromState
		}

		hasher := sha1.New()
		if err := c.msg.Read(io.TeeReader(c.msg.Buffer, hasher)); err != nil {
			return errorState("[bdat]: error %s", err)
		}

		c.PrintfLine("250 Ok : queued as +%x", hasher.Sum(nil))

		serverHandler{c.server}.Serve(*c.msg)
		/*
			if err = c.muxer.Handle(*c.msg); err != nil {
				return errorState(err.Error())
			}
		*/

		c.msg = c.newMessage()
		return loopState
	} else if isCommand(line, "DATA") {
		c.PrintfLine("354 Enter message, ending with \".\" on a line by itself")

		hasher := sha1.New()
		err := c.msg.Read(io.TeeReader(c.Text.DotReader(), hasher))
		if err != nil {
			return errorState("[data]: error %s", err)
		}

		c.PrintfLine("250 Ok : queued as +%x", hasher.Sum(nil))

		serverHandler{c.server}.Serve(*c.msg)

		/*
			if err = c.muxer.Handle(*c.msg); err != nil {
				return errorState(err.Error())
			}
		*/

		c.msg = c.newMessage()
		return loopState
	} else {
		return unrecognizedState
	}
}

func loopState(c *conn) stateFn {
	line, err := c.ReadLine()
	if err != nil {
		log.Error("[loopState] error: %s", err.Error())
		return nil
	}

	if line == "" {
		return loopState
	}

	c.i++

	if c.i > 100 {
		return errorState("Error: invalid.")
	}

	if isCommand(line, "MAIL FROM") {
		c.msg.From, _ = mail.ParseAddress(line[10:])
		c.PrintfLine("250 Ok")
		return mailFromState
	} else if isCommand(line, "STARTTLS") {
		if !c.server.starttls {
			return unrecognizedState
		}

		c.PrintfLine("220 Ready to start TLS")

		tlsConn := tls.Server(c.rwc, c.server.TLSConfig)

		if err := tlsConn.Handshake(); err != nil {
			log.Error("Error during tls handshake: %s", err.Error())
			return nil
		}

		c.Text = textproto.NewConn(tlsConn)
		return helloState
	} else if isCommand(line, "RSET") {
		c.msg = c.newMessage()
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

	if isCommand(line, "HELO") {
		domain, err := parseHelloArgument(line)
		if err != nil {
			return errorState(err.Error())
		}

		c.domain = domain
		c.msg.Domain = domain

		c.PrintfLine("250 Hello %s, I am glad to meet you", domain)
		return loopState
	} else if isCommand(line, "EHLO") {
		domain, err := parseHelloArgument(line)
		if err != nil {
			return errorState(err.Error())
		}

		c.domain = domain
		c.msg.Domain = domain

		c.PrintfLine("250-Hello %s", domain)
		c.PrintfLine("250-SIZE 35882577")
		c.PrintfLine("250-8BITMIME")
		if c.server.starttls {
			c.PrintfLine("250-STARTTLS")
		}
		c.PrintfLine("250-ENHANCEDSTATUSCODES")
		c.PrintfLine("250-PIPELINING")
		c.PrintfLine("250-CHUNKING")
		c.PrintfLine("250 SMTPUTF8")
		return loopState
	} else {
		return errorState("Before we shake hands it will be appropriate to tell me who you are.")
	}
}

func (c *conn) serve() {
	c.Text = textproto.NewConn(c.rwc)
	defer c.Text.Close()

	// todo add idle timeout here

	for state := startState; state != nil; {
		state = state(c)
	}
}
