package smtpd

import "testing"

func TestWithListener(t *testing.T) {
	l := Listener{
		Address: "127.0.0.1",
		Port:    "8025",
		Mode:    "tls",
	}

	cfg := &Config{}

	_ = WithListener(l)(cfg)

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
	l := Listener{
		Address: "127.0.0.1",
		Port:    "8025",
	}

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
