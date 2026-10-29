package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsAndOverrides(t *testing.T) {
	d := t.TempDir()
	c, e := Load(d)
	if e != nil || c.AdminPort != 8765 || c.PhonePort != 8088 || c.Limits.CSVBytes != 2<<20 {
		t.Fatal(c, e)
	}
	os.WriteFile(filepath.Join(d, "config.json"), []byte(`{"format":"phonebook","limits":{"contacts":50}}`), 0600)
	c, e = Load(d)
	if e != nil || c.Format != "phonebook" || c.Limits.Contacts != 50 || c.Limits.Number != 32 {
		t.Fatal(c, e)
	}
}
func TestInvalidConfiguration(t *testing.T) {
	for _, data := range []string{`{`, `null`, `{"adminPort":0}`, `{"adminPort":8088}`, `{"format":"bad"}`, `{"limits":{"contacts":0}}`, `{"allowedSubnet":"garbage"}`, `{"adminPort":8765,"unknown":1}`} {
		d := t.TempDir()
		os.WriteFile(filepath.Join(d, "config.json"), []byte(data), 0600)
		if _, e := Load(d); e == nil {
			t.Fatal("accepted", data)
		}
	}
}
