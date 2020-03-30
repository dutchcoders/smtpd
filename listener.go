package smtpd

import "crypto/tls"

type Listener struct {
	Address   string
	Port      string
	Mode      string //smtp modes: 'plain (25)', 'tls (465)' or 'starttls (587)'
	Banner    func() string
	TLSConfig *tls.Config
	Handler   Handler
}

func NewListener(options ...func(*Listener)) {
	l := &Listener{
		Banner: func() string { return "DutchCoders SMTPd" },
	}

	for _, opt := range options {
		opt(l)
	}
}

func ListenWithAddress(s string) func(*Listener) {
	return func(l *Listener) {
		l.Address = s
	}
}

func ListenWithPort(s string) func(*Listener) {
	return func(l *Listener) {
		l.Port = s
	}
}

func ListenWithMode(s string) func(*Listener) {
	return func(l *Listener) {
		l.Mode = s
	}
}

func ListenWithBanner(fn func() string) func(*Listener) {
	return func(l *Listener) {
		l.Banner = fn
	}
}

func ListenWithTLSConfig(tlsconf *tls.Config) func(*Listener) {
	return func(l *Listener) {
		l.TLSConfig = tlsconf
	}
}

func ListenWithHandler(h Handler) func(*Listener) {
	return func(l *Listener) {
		l.Handler = h
	}
}
