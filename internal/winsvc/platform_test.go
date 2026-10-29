package winsvc

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSubnetValidation(t *testing.T) {
	for _, s := range []string{"", "any", "0.0.0.0/0", "192.168.1.0", "::/0"} {
		if ValidateSubnet(s) == nil {
			t.Fatal("accepted", s)
		}
	}
	if e := ValidateSubnet("192.168.1.0/24"); e != nil {
		t.Fatal(e)
	}
}
func TestPurgeRequiresRegisteredDefaultDirectory(t *testing.T) {
	for _, args := range []struct {
		requested, registered, def string
		ok                         bool
	}{{"/default", "/custom", "/default", false}, {"/default", "/default", "/default", true}, {"/custom", "/custom", "/default", false}, {"/default", "", "/default", false}} {
		e := ValidatePurge(args.requested, args.registered, args.def)
		if (e == nil) != args.ok {
			t.Fatal(args, e)
		}
	}
}
func TestSecureDescriptorIncludesTrustedOwner(t *testing.T) {
	s := secureDescriptor(true)
	if !strings.HasPrefix(s, "O:BAG:BA") || !strings.Contains(s, "D:P") {
		t.Fatal(s)
	}
}

type fakeLifecycle struct {
	states                   []ServiceState
	stops, deletes, disabled int
	stopErr                  error
}

func (f *fakeLifecycle) State() (ServiceState, error) {
	s := f.states[0]
	if len(f.states) > 1 {
		f.states = f.states[1:]
	}
	return s, nil
}
func (f *fakeLifecycle) Stop() error            { f.stops++; return f.stopErr }
func (f *fakeLifecycle) Delete() error          { f.deletes++; return nil }
func (f *fakeLifecycle) DisableRecovery() error { f.disabled++; return nil }
func TestRollbackWaitsForStop(t *testing.T) {
	f := &fakeLifecycle{states: []ServiceState{StateStarting, StateRunning, StateStopping, StateStopped}}
	if e := rollbackService(f, time.Second); e != nil {
		t.Fatal(e)
	}
	if f.deletes != 1 || f.stops != 1 || f.disabled != 1 {
		t.Fatal(f)
	}
}
func TestRollbackReportsStopFailure(t *testing.T) {
	f := &fakeLifecycle{states: []ServiceState{StateRunning}, stopErr: errors.New("stop denied")}
	e := rollbackService(f, time.Second)
	if e == nil || !strings.Contains(e.Error(), "stop denied") || f.deletes != 0 {
		t.Fatal(e, f)
	}
}
