package smtpd

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/mail"

	"github.com/google/uuid"
)

type Message struct {
	MessageID uuid.UUID
	From      *mail.Address
	To        []*mail.Address

	Domain     string
	RemoteAddr string

	Header mail.Header
	Buffer *bytes.Buffer
	Body   *bytes.Buffer
}

func (c *conn) newMessage() *Message {
	id, _ := uuid.NewUUID()

	return &Message{
		MessageID:  id,
		To:         []*mail.Address{},
		Body:       &bytes.Buffer{},
		Buffer:     &bytes.Buffer{},
		Domain:     c.domain,
		RemoteAddr: c.RemoteAddr().String(),
	}
}

func (m *Message) Read(r io.Reader) error {
	buff, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	msg, err := mail.ReadMessage(bytes.NewReader(buff))
	if err != nil {
		m.Body = bytes.NewBuffer(buff)
		return err
	}

	m.Header = msg.Header

	buff, err = ioutil.ReadAll(msg.Body)
	if err != nil {
		return err
	}

	m.Body = bytes.NewBuffer(buff)
	return err
}
