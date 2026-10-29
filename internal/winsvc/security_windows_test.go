//go:build windows

package winsvc

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
)

func TestSecureDirOwnerAndProtectedDACL(t *testing.T) {
	if !windows.GetCurrentProcessToken().IsElevated() {
		t.Skip("requires elevated Windows acceptance host")
	}
	d := t.TempDir()
	p := filepath.Join(d, "book.json")
	if e := os.WriteFile(p, []byte("{}"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := SecureDir(d); e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{d, p} {
		sd, e := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
		if e != nil {
			t.Fatal(e)
		}
		owner, _, e := sd.Owner()
		if e != nil || owner.String() != "S-1-5-32-544" {
			t.Fatal("untrusted owner", owner, e)
		}
		control, _, e := sd.Control()
		if e != nil || control&windows.SE_DACL_PROTECTED == 0 {
			t.Fatal("DACL inherits", control, e)
		}
	}
}
