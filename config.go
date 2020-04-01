package smtpd

import (
	"fmt"
	"strconv"
)

type Config struct {
	Listeners []Listener
}

func WithListener(ll ...Listener) func(*Config) error {
	return func(cfg *Config) error {
		for n, l := range ll {
			if l.ID == "" {
				l.ID = strconv.Itoa(n)
			}

			if l.Mode == "" {
				l.Mode = "plain"
			}

			if l.Port == "" {
				return fmt.Errorf("[%s] Required field \"Port\" is empty!", l.ID)
			}
		}

		cfg.Listeners = append(cfg.Listeners, ll...)
		return nil
	}
}
