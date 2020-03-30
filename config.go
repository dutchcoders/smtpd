package smtpd

import (
	"crypto/tls"
	"fmt"
)

type Config struct {
	Listeners []Listener
	Banner    func() string
	TLSConfig *tls.Config
}

type Listener struct {
	Address string
	Port    string
	Mode    string //smtp modes: 'plain (25)', 'tls (465)' or 'starttls (587)'
}

func WithListener(l Listener) func(*Config) error {
	return func(cfg *Config) error {
		switch {
		case l.Address == "":
			return fmt.Errorf("Required field Listener.Address is empty!")
		case l.Port == "":
			return fmt.Errorf("Required field Listener.Port is empty!")
		case l.Mode == "":
			l.Mode = "plain"
		}
		cfg.Listeners = append(cfg.Listeners, l)
		return nil
	}
}

func TLSConfig(v *tls.Config) func(*Config) error {
	return func(cfg *Config) error {
		cfg.TLSConfig = v
		return nil
	}
}

func Banner(fn func() string) func(*Config) error {
	return func(cfg *Config) error {
		cfg.Banner = fn
		return nil
	}
}
