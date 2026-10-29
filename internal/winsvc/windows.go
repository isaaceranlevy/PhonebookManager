//go:build windows

package winsvc

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func IsService() (bool, error) { return svc.IsWindowsService() }

type handler struct {
	run func(context.Context, func()) error
}

func (h handler) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending, WaitHint: 15000}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	ready := make(chan struct{}, 1)
	go func() { done <- h.run(ctx, func() { ready <- struct{}{} }) }()
	current := svc.Status{State: svc.StartPending}
	for {
		select {
		case <-ready:
			current = svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
			status <- current
		case e := <-done:
			if e != nil {
				log.Printf("service stopped: %v", e)
				return false, 1
			}
			return false, 0
		case r := <-requests:
			switch r.Cmd {
			case svc.Interrogate:
				status <- current
			case svc.Stop, svc.Shutdown:
				current = svc.Status{State: svc.StopPending, WaitHint: 15000}
				status <- current
				cancel()
			}
		}
	}
}
func RunService(run func(context.Context, func()) error) error { return svc.Run(Name, handler{run}) }
func SecureDir(dir string) error {
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink in service data directory: %s", path)
		}
		sddl := secureDescriptor(d.IsDir())
		sd, e := windows.SecurityDescriptorFromString(sddl)
		if e != nil {
			return e
		}
		acl, _, e := sd.DACL()
		if e != nil {
			return e
		}
		owner, _, e := sd.Owner()
		if e != nil {
			return e
		}
		return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, owner, nil, acl, nil)
	})
}
func netsh(args ...string) error {
	out, e := exec.Command("netsh", args...).CombinedOutput()
	if e != nil {
		return fmt.Errorf("netsh: %w: %s", e, out)
	}
	return nil
}
func waitState(s *mgr.Service, state svc.State) error {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		v, e := s.Query()
		if e != nil {
			return e
		}
		if v.State == state {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("service did not reach state %d within 20 seconds", state)
}
func Manage(command, dir, subnet string, port int, purge bool) (err error) {
	m, e := mgr.Connect()
	if e != nil {
		return e
	}
	defer m.Disconnect()
	rule := Name + " XML"
	switch command {
	case "install":
		if e = ValidateSubnet(subnet); e != nil {
			return e
		}
		if e = SecureDir(dir); e != nil {
			return e
		}
		exe, e := os.Executable()
		if e != nil {
			return e
		}
		s, e := m.CreateService(Name, exe, mgr.Config{DisplayName: "Nimbus Phonebook", Description: "Local phonebook manager and read-only Yealink directory", StartType: mgr.StartAutomatic, DelayedAutoStart: true}, "run", "--data-dir", dir)
		if e != nil {
			return e
		}
		defer s.Close()
		firewall := false
		defer func() {
			if err != nil {
				if firewall {
					err = errors.Join(err, netsh("advfirewall", "firewall", "delete", "rule", "name="+rule))
				}
				err = errors.Join(err, rollbackService(serviceLifecycle{s}, 20*time.Second))
			}
		}()
		if e = s.SetRecoveryActions([]mgr.RecoveryAction{{Type: mgr.ServiceRestart, Delay: 5 * time.Second}, {Type: mgr.ServiceRestart, Delay: 15 * time.Second}, {Type: mgr.ServiceRestart, Delay: 60 * time.Second}}, 86400); e != nil {
			return e
		}
		if e = s.SetRecoveryActionsOnNonCrashFailures(true); e != nil {
			return e
		}
		if e = netsh("advfirewall", "firewall", "add", "rule", "name="+rule, "dir=in", "action=allow", "protocol=TCP", "localport="+strconv.Itoa(port), "remoteip="+subnet, "program="+exe, "profile=domain,private"); e != nil {
			return e
		}
		firewall = true
		if e = s.Start(); e != nil {
			return e
		}
		return waitState(s, svc.Running)
	case "status":
		s, e := m.OpenService(Name)
		if e != nil {
			return e
		}
		defer s.Close()
		v, e := s.Query()
		if e != nil {
			return e
		}
		names := map[svc.State]string{svc.Stopped: "Stopped", svc.StartPending: "StartPending", svc.StopPending: "StopPending", svc.Running: "Running"}
		fmt.Printf("%s: %s (%d)\n", Name, names[v.State], v.State)
		return nil
	case "uninstall":
		s, e := m.OpenService(Name)
		if e != nil {
			return e
		}
		defer s.Close()
		if purge {
			cfg, e := s.Config()
			if e != nil {
				return e
			}
			args, e := windows.DecomposeCommandLine(cfg.BinaryPathName)
			if e != nil {
				return e
			}
			registered := ""
			if len(args) == 4 && args[1] == "run" && args[2] == "--data-dir" {
				registered = args[3]
			}
			expected := filepath.Join(os.Getenv("ProgramData"), Name)
			if e = ValidatePurge(dir, registered, expected); e != nil {
				return e
			}
		}
		if e = stopService(serviceLifecycle{s}, 20*time.Second); e != nil {
			return e
		}
		if e = s.Delete(); e != nil {
			return e
		}
		if e = netsh("advfirewall", "firewall", "delete", "rule", "name="+rule); e != nil {
			return e
		}
		if purge {
			return os.RemoveAll(dir)
		}
		return nil
	}
	return fmt.Errorf("unknown command %q", command)
}

// Adapter keeps state/rollback policy testable without a Windows SCM.
type serviceLifecycle struct{ s *mgr.Service }

func (v serviceLifecycle) State() (ServiceState, error) {
	q, e := v.s.Query()
	if e != nil {
		return StateStopped, e
	}
	switch q.State {
	case svc.Stopped:
		return StateStopped, nil
	case svc.StartPending:
		return StateStarting, nil
	case svc.StopPending:
		return StateStopping, nil
	default:
		return StateRunning, nil
	}
}
func (v serviceLifecycle) Stop() error   { _, e := v.s.Control(svc.Stop); return e }
func (v serviceLifecycle) Delete() error { return v.s.Delete() }
func (v serviceLifecycle) DisableRecovery() error {
	c, e := v.s.Config()
	if e != nil {
		return e
	}
	c.StartType = windows.SERVICE_DISABLED
	if e = v.s.UpdateConfig(c); e != nil {
		return e
	}
	return v.s.ResetRecoveryActions()
}
