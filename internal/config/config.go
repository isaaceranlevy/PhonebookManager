package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/isaaceranlevy/PhonebookManager/internal/book"
	"io"
	"net"
	"os"
	"path/filepath"
)

type Config struct {
	AdminPort     int         `json:"adminPort"`
	PhonePort     int         `json:"phonePort"`
	Format        string      `json:"format"`
	Limits        book.Limits `json:"limits"`
	AllowedSubnet string      `json:"allowedSubnet"`
}

func Default() Config {
	return Config{AdminPort: 8765, PhonePort: 8088, Format: "directory", Limits: book.DefaultLimits()}
}
func Load(dir string) (Config, error) {
	c := Default()
	b, e := os.ReadFile(filepath.Join(dir, "config.json"))
	if os.IsNotExist(e) {
		return c, nil
	}
	if e != nil {
		return c, e
	}
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		return c, fmt.Errorf("config must be a JSON object")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(&c); e != nil {
		return c, e
	}
	if d.Decode(new(any)) != io.EOF {
		return c, fmt.Errorf("config must contain a single JSON object")
	}
	return c, c.Validate()
}
func (c Config) Validate() error {
	if c.AdminPort < 1 || c.AdminPort > 65535 || c.PhonePort < 1 || c.PhonePort > 65535 || c.AdminPort == c.PhonePort {
		return fmt.Errorf("invalid ports")
	}
	if c.Format != "directory" && c.Format != "phonebook" {
		return fmt.Errorf("invalid XML format")
	}
	if c.AllowedSubnet != "" {
		ip, _, e := net.ParseCIDR(c.AllowedSubnet)
		if e != nil || ip.To4() == nil {
			return fmt.Errorf("allowedSubnet must be IPv4 CIDR")
		}
	}
	l := c.Limits
	if l.Contacts < 1 || l.Name < 1 || l.Number < 1 || l.JSONBytes < 1024 || l.CSVBytes < 1024 || l.JSONBytes > 64<<20 || l.CSVBytes > 64<<20 {
		return fmt.Errorf("invalid limits")
	}
	return nil
}
