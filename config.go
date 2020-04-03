package smtpd

import (
	"fmt"
	"strconv"
)

type Config struct {
	Listeners []Listener
}

//WithListener takes 1 or more Listener structs to serve on.
func WithListener(ll ...Listener) func(*Config) error {
	return func(cfg *Config) error {
		if len(ll) == 0 {
			return fmt.Errorf("Got no listeners to configure.")
		}

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

			if l.Mode == "tls" || l.Mode == "starttls" {
				if l.TLSConfig == nil {
					return fmt.Errorf("[%s] Mode 'tls/starttls' requires a tls config.", l.ID)
				}
			}
		}

		cfg.Listeners = append(cfg.Listeners, ll...)
		return nil
	}
}
