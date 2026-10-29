package winsvc

import (
	"fmt"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const Name = "NimbusPhonebook"

func ValidateSubnet(s string) error {
	ip, n, e := net.ParseCIDR(s)
	if e != nil || ip.To4() == nil {
		return fmt.Errorf("--subnet requires an explicit IPv4 CIDR, for example 192.168.1.0/24")
	}
	ones, _ := n.Mask.Size()
	if ones < 8 {
		return fmt.Errorf("subnet must be /8 or narrower")
	}
	return nil
}

// Purge is deliberately restricted to the registered default data directory.
func ValidatePurge(requested, registered, def string) error {
	if registered == "" {
		return fmt.Errorf("registered service data directory is unknown; refusing purge")
	}
	paths := []string{requested, registered, def}
	for i, p := range paths {
		a, e := filepath.Abs(p)
		if e != nil {
			return e
		}
		paths[i] = filepath.Clean(a)
		if runtime.GOOS == "windows" {
			paths[i] = strings.ToLower(paths[i])
		}
	}
	if paths[0] != paths[2] || paths[1] != paths[2] {
		return fmt.Errorf("--purge requires both the requested and registered service data directories to be the default ProgramData directory; remove custom data manually")
	}
	return nil
}
func secureDescriptor(directory bool) string {
	if directory {
		return "O:BAG:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"
	}
	return "O:BAG:BAD:P(A;;FA;;;SY)(A;;FA;;;BA)"
}

type ServiceState int

const (
	StateStopped ServiceState = iota
	StateStarting
	StateRunning
	StateStopping
)

type lifecycle interface {
	State() (ServiceState, error)
	Stop() error
	Delete() error
	DisableRecovery() error
}

func stopService(s lifecycle, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	sent := false
	for {
		state, e := s.State()
		if e != nil {
			return e
		}
		if state == StateStopped {
			return nil
		}
		if state == StateRunning && !sent {
			if e = s.Stop(); e != nil {
				return fmt.Errorf("service stop failed: %w", e)
			}
			sent = true
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("service did not stop; it may still be running, inspect SCM before retrying")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
func rollbackService(s lifecycle, timeout time.Duration) error {
	if e := s.DisableRecovery(); e != nil {
		return fmt.Errorf("rollback disabling recovery failed; inspect SCM: %w", e)
	}
	if e := stopService(s, timeout); e != nil {
		return fmt.Errorf("installation rollback incomplete: %w", e)
	}
	if e := s.Delete(); e != nil {
		return fmt.Errorf("rollback deleting stopped service failed: %w", e)
	}
	return nil
}

// Users may read/execute the program and guides, but only SYSTEM/admins can modify them.
func installDescriptor(directory bool) string {
	if directory {
		return "O:BAG:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;GRGX;;;BU)"
	}
	return "O:BAG:BAD:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;GRGX;;;BU)"
}
