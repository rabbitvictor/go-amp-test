package config

import "testing"

func TestDBConfig_DSN(t *testing.T) {
	c := DBConfig{
		Path:         "app.db",
		MaxOpenConns: 1,
		BusyTimeout:  5000,
		JournalMode:  "WAL",
		Synchronous:  "NORMAL",
		ForeignKeys:  true,
	}
	want := "file:app.db?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_foreign_keys=on"
	if got := c.DSN(); got != want {
		t.Errorf("DSN = %q, want %q", got, want)
	}
}

func TestDBConfig_DSN_ForeignKeysOff(t *testing.T) {
	c := DBConfig{
		Path:        ":memory:",
		BusyTimeout: 1000,
		JournalMode: "MEMORY",
		Synchronous: "OFF",
		ForeignKeys: false,
	}
	want := "file::memory:?_journal_mode=MEMORY&_busy_timeout=1000&_synchronous=OFF&_foreign_keys=off"
	if got := c.DSN(); got != want {
		t.Errorf("DSN = %q, want %q", got, want)
	}
}

func TestServerConfig_Addr(t *testing.T) {
	s := ServerConfig{Port: "8080"}
	if got := s.Addr(); got != ":8080" {
		t.Errorf("Addr = %q, want %q", got, ":8080")
	}
}
