package store

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/isaaceranlevy/PhonebookManager/internal/book"
	"github.com/isaaceranlevy/PhonebookManager/internal/yealink"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var ErrConflict = errors.New("הספר השתנה. יש לטעון מחדש לפני שמירה")

type Snapshot struct {
	Book     book.Book
	XML      []byte
	ETag     string
	Degraded bool
}
type Store struct {
	mu      sync.Mutex
	current atomic.Pointer[Snapshot]
	dir     string
	limits  book.Limits
	format  string
}
type Backup struct {
	ID        string    `json:"id"`
	Revision  int64     `json:"revision"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func clone(v *Snapshot) *Snapshot {
	r := *v
	r.XML = append([]byte(nil), v.XML...)
	r.Book.Contacts = make([]book.Contact, len(v.Book.Contacts))
	for i, c := range v.Book.Contacts {
		r.Book.Contacts[i] = c
		r.Book.Contacts[i].Phones = append([]string(nil), c.Phones...)
	}
	return &r
}
func (s *Store) Snapshot() *Snapshot { return clone(s.current.Load()) }
func (s *Store) read(path string) (book.Book, error) {
	var b book.Book
	data, e := os.ReadFile(path)
	if e != nil {
		return b, e
	}
	if e = json.Unmarshal(data, &b); e != nil {
		return b, e
	}
	if b.Contacts == nil || b.UpdatedAt.IsZero() || b.Revision < 1 {
		return b, fmt.Errorf("invalid book structure, revision or timestamp")
	}
	if e = book.Validate(b.Contacts, s.limits); e != nil {
		return b, e
	}
	if b.Contacts == nil {
		b.Contacts = []book.Contact{}
	}
	return b, nil
}
func (s *Store) makeSnapshot(b book.Book, degraded bool) (*Snapshot, error) {
	x, e := yealink.Encode(b.Contacts, s.format)
	if e != nil {
		return nil, e
	}
	return &Snapshot{b, x, fmt.Sprintf("\"%x\"", sha256.Sum256(x)), degraded}, nil
}
func Open(dir string, l book.Limits, format string) (*Store, error) {
	s := &Store{dir: dir, limits: l, format: format}
	if e := os.MkdirAll(filepath.Join(dir, "backups"), 0700); e != nil {
		return nil, e
	}
	b, e := s.read(filepath.Join(dir, "book.json"))
	degraded := false
	if e != nil {
		primaryErr := e
		bs, backupErr := s.Backups()
		if backupErr != nil {
			return nil, backupErr
		}
		found := false
		for _, v := range bs {
			b, e = s.read(filepath.Join(dir, "backups", v.ID))
			if e == nil {
				found = true
				break
			}
		}
		if found {
			degraded = true
		} else if errors.Is(primaryErr, os.ErrNotExist) && len(bs) == 0 {
			entries, err := os.ReadDir(filepath.Join(dir, "backups"))
			if err != nil {
				return nil, err
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), "book-r") {
					return nil, fmt.Errorf("primary missing and backups invalid")
				}
			}
			b = book.Book{Contacts: []book.Contact{}}
		} else {
			return nil, fmt.Errorf("book.json unavailable and no valid backup: %w", primaryErr)
		}
	}
	v, e := s.makeSnapshot(b, degraded)
	if e != nil {
		return nil, e
	}
	s.current.Store(v)
	return s, nil
}
func (s *Store) Save(rev int64, cs []book.Contact) (book.Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.current.Load()
	if old.Book.Revision != rev {
		return book.Book{}, ErrConflict
	}
	if e := book.Validate(cs, s.limits); e != nil {
		return book.Book{}, e
	}
	if cs == nil {
		cs = []book.Contact{}
	}
	b := book.Book{Revision: rev + 1, UpdatedAt: time.Now().UTC(), Contacts: cs}
	next, e := s.makeSnapshot(b, false)
	if e != nil {
		return book.Book{}, e
	}
	next = clone(next)
	// Preserve only a verified primary: recovery must never overwrite valid backups with corrupt bytes.
	path := filepath.Join(s.dir, "book.json")
	if _, e := s.read(path); e == nil {
		data, e := os.ReadFile(path)
		if e != nil {
			return book.Book{}, e
		}
		id := fmt.Sprintf("book-r%020d-%s.json", old.Book.Revision, time.Now().UTC().Format("20060102T150405.000000000Z"))
		if e = os.WriteFile(filepath.Join(s.dir, "backups", id), data, 0600); e != nil {
			return book.Book{}, e
		}
	} else if !errors.Is(e, os.ErrNotExist) && !old.Degraded {
		return book.Book{}, e
	}
	data, e := json.MarshalIndent(next.Book, "", "  ")
	if e != nil {
		return book.Book{}, e
	}
	if int64(len(data)) > s.limits.JSONBytes {
		return book.Book{}, &book.ValidationError{Errors: []string{"הספר גדול ממגבלת JSON"}}
	}
	tmp := path + ".tmp"
	f, e := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if e != nil {
		return book.Book{}, e
	}
	if _, e = f.Write(data); e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		os.Remove(tmp)
		return book.Book{}, e
	}
	if e = replaceFile(tmp, path); e != nil {
		os.Remove(tmp)
		return book.Book{}, e
	}
	s.current.Store(next)
	// Rotation is maintenance after the commit; its failure must not turn a committed save into a failed response.
	s.rotate()
	return s.Snapshot().Book, nil
}
func (s *Store) Backups() ([]Backup, error) {
	files, e := os.ReadDir(filepath.Join(s.dir, "backups"))
	if e != nil {
		return nil, e
	}
	result := []Backup{}
	for _, f := range files {
		if f.IsDir() || !strings.HasPrefix(f.Name(), "book-r") || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		b, e := s.read(filepath.Join(s.dir, "backups", f.Name()))
		if e == nil {
			result = append(result, Backup{f.Name(), b.Revision, b.UpdatedAt})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Revision != result[j].Revision {
			return result[i].Revision > result[j].Revision
		}
		return result[i].ID > result[j].ID
	})
	return result, nil
}
func (s *Store) rotate() {
	bs, e := s.Backups()
	if e != nil {
		return
	}
	for i := 10; i < len(bs); i++ {
		os.Remove(filepath.Join(s.dir, "backups", bs[i].ID))
	}
}
func (s *Store) Restore(id string, rev int64) (book.Book, error) {
	if filepath.Base(id) != id || !strings.HasPrefix(id, "book-r") || !strings.HasSuffix(id, ".json") {
		return book.Book{}, &book.ValidationError{Errors: []string{"מזהה גיבוי לא תקין"}}
	}
	b, e := s.read(filepath.Join(s.dir, "backups", id))
	if e != nil {
		return b, e
	}
	return s.Save(rev, b.Contacts)
}
