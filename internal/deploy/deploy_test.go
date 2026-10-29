package deploy

import (
	"encoding/binary"
	"github.com/isaaceranlevy/PhonebookManager/internal/config"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf16"
)

func TestNetworkSuggestions(t *testing.T) {
	got := Subnets([]net.Addr{&net.IPNet{IP: net.ParseIP("192.168.1.10"), Mask: net.CIDRMask(24, 32)}, &net.IPNet{IP: net.ParseIP("192.168.1.11"), Mask: net.CIDRMask(24, 32)}, &net.IPNet{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)}, &net.IPNet{IP: net.ParseIP("169.254.1.2"), Mask: net.CIDRMask(16, 32)}, &net.IPNet{IP: net.ParseIP("2001:db8::1"), Mask: net.CIDRMask(64, 128)}})
	if len(got) != 1 || got[0] != "192.168.1.0/24" {
		t.Fatal(got)
	}
}
func TestSubnetChoice(t *testing.T) {
	for _, x := range []struct {
		saved string
		nets  []string
		want  string
	}{{"", []string{"192.168.1.0/24"}, "192.168.1.0/24"}, {"", []string{"192.168.1.0/24", "10.0.0.0/24"}, ""}, {"10.0.0.0/24", []string{"192.168.1.0/24"}, "10.0.0.0/24"}} {
		if s := DefaultSubnet(x.saved, x.nets); s != x.want {
			t.Fatal(s, x)
		}
	}
}
func TestValidateExistingConfigBeforeChange(t *testing.T) {
	d := t.TempDir()
	data := `{"adminPort":9876,"phonePort":8888,"allowedSubnet":"192.168.1.0/24"}`
	os.WriteFile(filepath.Join(d, "config.json"), []byte(data), 0600)
	if e := ValidateConfig(d, "10.0.0.0/24"); e == nil {
		t.Fatal("silently changed subnet")
	}
	if e := ValidateConfig(d, "192.168.1.0/24"); e != nil {
		t.Fatal(e)
	}
	after, _ := os.ReadFile(filepath.Join(d, "config.json"))
	if string(after) != data {
		t.Fatal("mutated config")
	}
	if e := ValidateConfig(t.TempDir(), "0.0.0.0/0"); e == nil {
		t.Fatal("invalid subnet")
	}
}
func TestRegistrationOwnership(t *testing.T) {
	for _, x := range []struct {
		args []string
		ok   bool
	}{{[]string{"/app/phonebook.exe", "run", "--data-dir", "/data"}, true}, {[]string{"/other/phonebook.exe", "run", "--data-dir", "/data"}, false}, {[]string{"/app/phonebook.exe", "run", "--data-dir", "/custom"}, false}, {[]string{"/app/phonebook.exe", "run"}, false}} {
		e := ValidateRegistration(x.args, "/app/phonebook.exe", "/data")
		if (e == nil) != x.ok {
			t.Fatal(x, e)
		}
	}
}
func TestInstallerInfoEncoding(t *testing.T) {
	data := InfoINI(config.Default(), []string{"192.168.1.0/24", "10.0.0.0/24"})
	if len(data) < 2 || data[0] != 255 || data[1] != 254 {
		t.Fatal("missing UTF-16 BOM")
	}
	text := DecodeINIForTest(data)
	if !strings.Contains(text, "AdminPort=8765") || !strings.Contains(text, "Count=2") || !strings.Contains(text, "Subnet=\r\n") {
		t.Fatal(text)
	}
}

func DecodeINIForTest(data []byte) string {
	var chars []uint16
	for i := 2; i < len(data); i += 2 {
		chars = append(chars, binary.LittleEndian.Uint16(data[i:]))
	}
	return string(utf16.Decode(chars))
}
func TestFixedProgramFilesInstallPath(t *testing.T) {
	for _, p := range []string{"/users/public/Nimbus", "/programs/Other", "/programs/NimbusPhonebook/..", "/programs/NimbusPhonebook/extra"} {
		if ValidateInstallPath(p, "/programs") == nil {
			t.Fatal("accepted writable/custom destination", p)
		}
	}
	if e := ValidateInstallPath("/programs/NimbusPhonebook", "/programs"); e != nil {
		t.Fatal(e)
	}
}
func TestReadyRequiresNimbusResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("a different web server")) }))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	if e := WaitReady(port, 0); e == nil {
		t.Fatal("accepted unrelated HTTP 200")
	}
}
func TestReadyRetriesUntilAPIAvailable(t *testing.T) {
	var count atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/info" {
			t.Error(r.URL.Path)
		}
		if count.Add(1) == 1 {
			w.WriteHeader(503)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"revision":1,"format":"directory"}`))
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	if e := WaitReady(port, time.Second); e != nil {
		t.Fatal(e)
	}
	if count.Load() < 2 {
		t.Fatal("did not retry")
	}
}
