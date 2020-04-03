package smtpd

import (
	"crypto/tls"
	"testing"
)

func TestWithListenerTLSNoConfig(t *testing.T) {
	l := Listener{
		Port: "8025",
		Mode: "tls",
	}

	err := WithListener(l)(&Config{})
	if err == nil {
		t.Error("Mode 'tls' without tls config did not return an error.")
	}

	l.Mode = "starttls"

	err = WithListener(l)(&Config{})
	if err == nil {
		t.Error("Mode 'starttls' without tls config did not return an error.")
	}
}

func TestWithListenerTLSAndConfig(t *testing.T) {
	l := Listener{
		Port:      "8025",
		TLSConfig: &tls.Config{},
		Mode:      "tls",
	}

	err := WithListener(l)(&Config{})
	if err != nil {
		t.Errorf("Mode 'tls' with tls config gives error: %v", err)
	}

	l.Mode = "starttls"

	err = WithListener(l)(&Config{})
	if err != nil {
		t.Errorf("Mode 'starttls' with tls config gives error: %v", err)
	}
}

func TestWithListener(t *testing.T) {
	l := Listener{
		ID:        "test",
		Address:   "127.0.0.1",
		Port:      "8025",
		Mode:      "tls",
		Banner:    func() string { return "test" },
		TLSConfig: &tls.Config{},
		Handler:   DefaultServeMux,
	}

	cfg := &Config{}

	err := WithListener(l)(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := cfg.Listeners

	if len(got) == 0 {
		t.Fatalf("Listener not added to Config")
	}

	if got[0].Address != l.Address {
		t.Errorf("Listener.Address got %s, want %s", got[0].Address, l.Address)
	}

	if got[0].Port != l.Port {
		t.Errorf("Listener.Address got %s, want %s", got[0].Port, l.Port)
	}

	if got[0].Mode != l.Mode {
		t.Errorf("Listener.Address got %s, want %s", got[0].Mode, l.Mode)
	}

	if got[0].ID != l.ID {
		t.Errorf("Listener.ID got %s, want %s", got[0].ID, l.ID)
	}

	if s := got[0].Banner(); s != "test" {
		t.Errorf("Listener.Banner got %s, want %s", s, "test")
	}

	if got[0].TLSConfig == nil {
		t.Error("Listener.TLSConfig got <nil>")
	}

	if got[0].Handler != DefaultServeMux {
		t.Error("Listener.Handler is not DefaultServeMux")
	}
}

func TestWithListenerError(t *testing.T) {
	l := Listener{}

	cfg := &Config{}

	err := WithListener(l)(cfg)

	if err == nil {
		t.Errorf("expected an error got none")
	}
}

func TestWithListenerDefaultMode(t *testing.T) {
	l := NewListener(ListenWithPort("25"))

	cfg := &Config{}

	_ = WithListener(l)(cfg)

	got := cfg.Listeners

	if len(got) == 0 {
		t.Fatalf("Listener not added to Config")
	}

	if got[0].Mode != "plain" {
		t.Errorf("Empty Listener.Mode should default to 'plain', got '%s'", got[0].Mode)
	}
}
