package main

import (
	"fmt"
	"github.com/isaaceranlevy/PhonebookManager/internal/config"
	"github.com/isaaceranlevy/PhonebookManager/internal/deploy"
	"github.com/isaaceranlevy/PhonebookManager/internal/winsvc"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func setupCommand(command, dir, installDir, subnet, output string) error {
	if command != "setup-info" && runtime.GOOS != "windows" {
		return fmt.Errorf("%s requires Windows", command)
	}
	args, exists, e := winsvc.RegistrationArgs()
	if e != nil {
		return e
	}
	if command == "setup-info" {
		if output == "" {
			return fmt.Errorf("--output is required")
		}
		return deploy.WriteInfo(dir, output, exists)
	}
	if installDir == "" {
		return fmt.Errorf("--install-dir is required")
	}
	programFiles := os.Getenv("ProgramW6432")
	if programFiles == "" {
		programFiles = os.Getenv("ProgramFiles")
	}
	if e = deploy.ValidateInstallPath(installDir, programFiles); e != nil {
		return e
	}
	exe := filepath.Join(installDir, "phonebook.exe")
	if exists {
		if e = deploy.ValidateRegistration(args, exe, dir); e != nil {
			return e
		}
	}
	if command == "setup-harden" {
		return winsvc.SecureInstallDir(installDir)
	}
	if command == "setup-remove" {
		if !exists {
			return nil
		}
		return winsvc.Manage("uninstall", dir, "", 0, false)
	}
	if e = deploy.ValidateConfig(dir, subnet); e != nil {
		return e
	}
	c, e := config.Load(dir)
	if e != nil {
		return e
	}
	checkPorts := func() error {
		for _, p := range []int{c.AdminPort, c.PhonePort} {
			l, e := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", p))
			if e != nil {
				return fmt.Errorf("port %d is busy; close any console instance of Nimbus or contact support: %w", p, e)
			}
			l.Close()
		}
		return nil
	}
	switch command {
	case "setup-ready":
		if !exists {
			return fmt.Errorf("service is not installed")
		}
		return deploy.WaitReady(c.AdminPort, 20*time.Second)
	case "setup-check":
		if !exists {
			return checkPorts()
		}
		return nil
	case "setup-prepare":
		if !exists {
			return checkPorts()
		}
		return winsvc.Manage("uninstall", dir, "", 0, false)
	case "setup-finish":
		current, e := os.Executable()
		if e != nil {
			return e
		}
		if e = deploy.ValidateRegistration([]string{current, "run", "--data-dir", dir}, exe, dir); e != nil {
			return e
		}
		if exists {
			if e = winsvc.StartExisting(); e != nil {
				return e
			}
		} else if e = execute([]string{"install", "--data-dir", dir, "--subnet", subnet}); e != nil {
			return e
		}
		return deploy.WaitReady(c.AdminPort, 20*time.Second)
	}
	return fmt.Errorf("unknown setup command")
}
