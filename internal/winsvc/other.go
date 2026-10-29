//go:build !windows

package winsvc

import (
	"context"
	"fmt"
	"os"
)

func IsService() (bool, error) { return false, nil }
func RunService(run func(context.Context, func()) error) error {
	return fmt.Errorf("Windows service commands require Windows")
}
func Manage(command, dir, subnet string, port int, purge bool) error {
	return fmt.Errorf("%s requires Windows", command)
}
func SecureDir(dir string) error                { return os.MkdirAll(dir, 0700) }
func RegistrationArgs() ([]string, bool, error) { return nil, false, nil }
func StartExisting() error                      { return fmt.Errorf("requires Windows") }
func SecureInstallDir(dir string) error         { return fmt.Errorf("requires Windows") }
