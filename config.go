package smtpd

import (
	"fmt"
)

type Config struct {
	Listeners []Listener
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
