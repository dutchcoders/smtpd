package smtpd

import (
	"crypto/tls"
)

type Config struct {
	ListenAddr string
	Banner     func() string
	TLSConfig  *tls.Config
}

func ListenAddr(v string) func(*Config) {
	return func(cfg *Config) {
		cfg.ListenAddr = v
	}
}

func TLSConfig(v *tls.Config) func(*Config) {
	return func(cfg *Config) {
		cfg.TLSConfig = v
	}
}

func Banner(fn func() string) func(*Config) {
	return func(cfg *Config) {
		cfg.Banner = fn
	}
}
