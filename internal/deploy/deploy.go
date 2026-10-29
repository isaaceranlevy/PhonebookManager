// Package deploy contains installer policy that can be tested without a Windows SCM.
package deploy

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/isaaceranlevy/PhonebookManager/internal/config"
	"github.com/isaaceranlevy/PhonebookManager/internal/winsvc"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf16"
)

func Subnets(addrs []net.Addr) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, a := range addrs {
		ip, n, e := net.ParseCIDR(a.String())
		if e != nil || ip.To4() == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() || !ip.IsGlobalUnicast() {
			continue
		}
		ones, bits := n.Mask.Size()
		if bits != 32 || ones < 8 {
			continue
		}
		s := n.String()
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
func Networks() ([]string, error) {
	ifs, e := net.Interfaces()
	if e != nil {
		return nil, e
	}
	var addrs []net.Addr
	for _, i := range ifs {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		as, e := i.Addrs()
		if e != nil {
			return nil, e
		}
		addrs = append(addrs, as...)
	}
	return Subnets(addrs), nil
}
func DefaultSubnet(saved string, nets []string) string {
	if saved != "" {
		return saved
	}
	if len(nets) == 1 {
		return nets[0]
	}
	return ""
}
func ValidateConfig(dir, subnet string) error {
	if e := winsvc.ValidateSubnet(subnet); e != nil {
		return e
	}
	c, e := config.Load(dir)
	if e != nil {
		return fmt.Errorf("existing configuration is invalid; contact support: %w", e)
	}
	if _, e = os.Stat(filepath.Join(dir, "config.json")); e == nil && c.AllowedSubnet != subnet {
		return fmt.Errorf("existing phone subnet is %q; select the same subnet or contact support", c.AllowedSubnet)
	} else if e != nil && !os.IsNotExist(e) {
		return e
	}
	return nil
}
func samePath(a, b string) bool {
	x, e := filepath.Abs(a)
	if e != nil {
		return false
	}
	y, e := filepath.Abs(b)
	if e != nil {
		return false
	}
	x, y = filepath.Clean(x), filepath.Clean(y)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(x, y)
	}
	return x == y
}
func ValidateRegistration(args []string, exe, dir string) error {
	if len(args) != 4 || !samePath(args[0], exe) || args[1] != "run" || args[2] != "--data-dir" || !samePath(args[3], dir) {
		return fmt.Errorf("existing NimbusPhonebook service uses another executable or data directory; contact support before changing it")
	}
	return nil
}
func InfoINI(c config.Config, nets []string) []byte {
	var text strings.Builder
	fmt.Fprintf(&text, "[Setup]\r\nSubnet=%s\r\nAdminPort=%d\r\nPhonePort=%d\r\nCount=%d\r\n", DefaultSubnet(c.AllowedSubnet, nets), c.AdminPort, c.PhonePort, len(nets))
	for i, s := range nets {
		fmt.Fprintf(&text, "Network%d=%s\r\n", i, s)
	}
	runes := utf16.Encode([]rune(text.String()))
	out := make([]byte, 2+2*len(runes))
	out[0], out[1] = 255, 254
	for i, r := range runes {
		binary.LittleEndian.PutUint16(out[2+i*2:], r)
	}
	return out
}
func WriteInfo(dir, path string, installed bool) error {
	c, e := config.Load(dir)
	if e != nil {
		return e
	}
	nets, e := Networks()
	if e != nil {
		return e
	}
	data := InfoINI(c, nets)
	flag := "ServiceExists=0\r\n"
	if installed {
		flag = "ServiceExists=1\r\n"
	}
	for _, r := range utf16.Encode([]rune(flag)) {
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], r)
		data = append(data, b[:]...)
	}
	return os.WriteFile(path, data, 0600)
}
func WaitReady(port int, timeout time.Duration) error {
	client := http.Client{Timeout: time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	deadline := time.Now().Add(timeout)
	for {
		r, e := client.Get(fmt.Sprintf("http://127.0.0.1:%d/api/info", port))
		if e == nil {
			var info struct {
				Revision *int64 `json:"revision"`
				Format   string `json:"format"`
			}
			valid := r.StatusCode == 200 && json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&info) == nil && info.Revision != nil && *info.Revision >= 0 && (info.Format == "directory" || info.Format == "phonebook")
			r.Body.Close()
			if valid {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("service did not become ready on loopback port %d; contact support and check logs", port)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func ValidateInstallPath(dir, programFiles string) error {
	if programFiles == "" || !samePath(dir, filepath.Join(programFiles, "NimbusPhonebook")) {
		return fmt.Errorf("installer only supports the fixed NimbusPhonebook directory in 64-bit Program Files")
	}
	return nil
}
