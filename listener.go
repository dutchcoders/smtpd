package smtpd

import "crypto/tls"

type Listener struct {
	//ID optional text to identify the listener in the logs.
	ID string

	//Address optional network address to listen on. Default: localhost
	Address string

	//Port required port to listen on.
	Port string

	//Mode optional smtp mode to use.
	//smtp modes: 'plain (25)', 'tls (465)' or 'starttls (587)'
	Mode string

	//Banner optional function returning the banner text shown to clients.
	Banner func() string

	//TLSConfig optional tls configuration.
	//note: mode 'tls' and 'starttls' require one,
	//  in mode 'plain' STARTTLS command will not be available without a config.
	TLSConfig *tls.Config

	//Handler optional handler(s) for this listener. Default: DefaultHandler
	Handler Handler
}

func NewListener(options ...func(*Listener)) Listener {
	l := Listener{
		Mode:   "plain",
		Banner: func() string { return "DutchCoders SMTPd" },
	}

	for _, opt := range options {
		opt(&l)
	}
	return l
}

func ListenWithID(id string) func(*Listener) {
	return func(l *Listener) {
		l.ID = id
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
