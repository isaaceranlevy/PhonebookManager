package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/isaaceranlevy/PhonebookManager/internal/book"
	"github.com/isaaceranlevy/PhonebookManager/internal/config"
	"github.com/isaaceranlevy/PhonebookManager/internal/server"
	"github.com/isaaceranlevy/PhonebookManager/internal/store"
	"github.com/isaaceranlevy/PhonebookManager/internal/winsvc"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func defaultDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("ProgramData"), "NimbusPhonebook")
	}
	return "data"
}
func main() {
	if e := execute(os.Args[1:]); e != nil {
		log.Print(e)
		os.Exit(1)
	}
}
func execute(args []string) error {
	command := "run"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}
	if command == "help" || command == "--help" || command == "-h" {
		fmt.Println("Nimbus Phonebook\n  run [--data-dir PATH]\n  serve-fixture [--format directory|phonebook]\n  install --subnet IPv4/CIDR [--data-dir PATH]\n  status\n  uninstall [--data-dir PATH] [--purge]\nWindows install/uninstall require Administrator privileges.")
		return nil
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	dir := fs.String("data-dir", defaultDir(), "data directory")
	subnet := fs.String("subnet", "", "allowed IPv4 CIDR for Windows firewall")
	format := fs.String("format", "", "Yealink fixture format")
	installDir := fs.String("install-dir", "", "installer target directory")
	output := fs.String("output", "", "installer metadata file")
	purge := fs.Bool("purge", false, "remove default ProgramData on uninstall")
	if e := fs.Parse(args); e != nil {
		return e
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	switch command {
	case "run", "serve-fixture", "install", "uninstall", "status", "setup-info", "setup-check", "setup-prepare", "setup-finish", "setup-remove", "setup-ready", "setup-harden":
	default:
		return fmt.Errorf("unknown command %q; use help", command)
	}
	abs, e := filepath.Abs(*dir)
	if e != nil {
		return e
	}
	*dir = abs
	if strings.HasPrefix(command, "setup-") {
		return setupCommand(command, *dir, *installDir, *subnet, *output)
	}
	if command == "install" || command == "uninstall" || command == "status" {
		if runtime.GOOS != "windows" {
			return fmt.Errorf("%s requires Windows", command)
		}
		if command != "install" {
			return winsvc.Manage(command, *dir, *subnet, 0, *purge)
		}
	}
	if command == "serve-fixture" {
		d, e := os.MkdirTemp("", "nimbus-fixture-")
		if e != nil {
			return e
		}
		defer os.RemoveAll(d)
		*dir = d
	}
	c, e := config.Load(*dir)
	if e != nil {
		return e
	}
	if *format != "" {
		if command != "serve-fixture" {
			return fmt.Errorf("--format is only for serve-fixture; use config.json for persistent configuration")
		}
		c.Format = *format
	}
	if e = c.Validate(); e != nil {
		return e
	}
	if command == "install" {
		if e = winsvc.ValidateSubnet(*subnet); e != nil {
			return e
		}
		if _, statErr := os.Stat(filepath.Join(*dir, "config.json")); statErr == nil && c.AllowedSubnet != *subnet {
			return fmt.Errorf("config allowedSubnet differs from --subnet")
		}
		if e = winsvc.SecureDir(*dir); e != nil {
			return e
		}
		c.AllowedSubnet = *subnet
		path := filepath.Join(*dir, "config.json")
		if _, e = os.Stat(path); os.IsNotExist(e) {
			b, _ := json.MarshalIndent(c, "", "  ")
			if e = os.WriteFile(path, b, 0600); e != nil {
				return e
			}
		} else if e != nil {
			return e
		}
	}
	if command == "install" || command == "uninstall" || command == "status" {
		return winsvc.Manage(command, *dir, *subnet, c.PhonePort, *purge)
	}
	run := func(ctx context.Context, ready func()) error {
		if e = winsvc.SecureDir(*dir); e != nil {
			return e
		}
		logs := filepath.Join(*dir, "logs")
		if e = os.MkdirAll(logs, 0700); e != nil {
			return e
		}
		f, e := os.OpenFile(filepath.Join(logs, "phonebook-"+time.Now().UTC().Format("2006-01-02")+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if e != nil {
			return e
		}
		defer f.Close()
		log.SetOutput(io.MultiWriter(os.Stderr, f))
		s, e := store.Open(*dir, c.Limits, c.Format)
		if e != nil {
			return e
		}
		if s.Snapshot().Degraded {
			log.Print("WARNING: corrupt book.json; recovered latest valid backup")
		}
		if command == "serve-fixture" {
			_, e = s.Save(0, []book.Contact{{ID: "fixture-1", Name: "בדיקת עברית & Nimbus", Phones: []string{"0012345", "+972501234567", "*123"}}, {ID: "fixture-2", Name: "תמיכה <משרד>", Phones: []string{"03-0123456"}}})
			if e != nil {
				return e
			}
		}
		a, e := server.New(s, c)
		if e != nil {
			return e
		}
		return a.RunReady(ctx, ready)
	}
	service, e := winsvc.IsService()
	if e != nil {
		return e
	}
	if service {
		return winsvc.RunService(run)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return run(ctx, func() {})
}
