package smtpd

import (
	"crypto/tls"
)

type Config struct {
	ListenAddr string
	Banner     func() string
	TLSConfig  *tls.Config
}

func ListenAddr(v string) func(*Config) error {
	return func(cfg *Config) error {
		cfg.ListenAddr = v
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
