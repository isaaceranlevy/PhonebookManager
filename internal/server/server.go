package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/isaaceranlevy/PhonebookManager/internal/book"
	"github.com/isaaceranlevy/PhonebookManager/internal/config"
	"github.com/isaaceranlevy/PhonebookManager/internal/store"
	"github.com/isaaceranlevy/PhonebookManager/web"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type App struct {
	Admin  http.Handler
	Phones http.Handler
	Token  string
	s      *store.Store
	c      config.Config
	subnet *net.IPNet
}

func New(s *store.Store, c config.Config) (*App, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	var random [32]byte
	if _, e := rand.Read(random[:]); e != nil {
		return nil, e
	}
	a := &App{s: s, c: c, Token: hex.EncodeToString(random[:])}
	if c.AllowedSubnet != "" {
		_, n, e := net.ParseCIDR(c.AllowedSubnet)
		if e != nil {
			return nil, e
		}
		a.subnet = n
	}
	a.Admin = http.HandlerFunc(a.admin)
	a.Phones = http.HandlerFunc(a.phones)
	return a, nil
}
func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, e error) {
	v := map[string]any{"error": e.Error()}
	var validation *book.ValidationError
	if errors.As(e, &validation) {
		v["errors"] = validation.Errors
	}
	jsonResponse(w, status, v)
}
func (a *App) result(w http.ResponseWriter, b book.Book, e error) {
	if e != nil {
		switch {
		case errors.Is(e, store.ErrConflict):
			fail(w, 409, e)
		default:
			var v *book.ValidationError
			if errors.As(e, &v) {
				fail(w, 400, e)
			} else {
				log.Printf("storage operation failed: %v", e)
				fail(w, 500, fmt.Errorf("השמירה נכשלה. הנתונים הקודמים נשמרו; בדקו את יומן השירות"))
			}
		}
		return
	}
	jsonResponse(w, 200, b)
}
func decode(w http.ResponseWriter, r *http.Request, v any, limit int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	e := d.Decode(v)
	if e == nil {
		if d.Decode(new(any)) != io.EOF {
			e = fmt.Errorf("נדרש מסמך JSON יחיד")
		}
	}
	if e != nil {
		code := 400
		var large *http.MaxBytesError
		if errors.As(e, &large) {
			code = 413
		}
		fail(w, code, e)
		return false
	}
	return true
}
func (a *App) admin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	port := strconv.Itoa(a.c.AdminPort)
	if r.Host != "127.0.0.1:"+port && r.Host != "localhost:"+port {
		fail(w, 403, fmt.Errorf("Host אסור"))
		return
	}
	if o := r.Header.Get("Origin"); o != "" && o != "http://"+r.Host {
		fail(w, 403, fmt.Errorf("Origin אסור"))
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Phonebook-Token")), []byte(a.Token)) != 1 {
			fail(w, 403, fmt.Errorf("אסימון חסר או פג תוקף; רעננו את הדף"))
			return
		}
		mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if mt != "application/json" {
			fail(w, 415, fmt.Errorf("נדרש application/json"))
			return
		}
	}
	switch r.URL.Path {
	case "/api/book":
		switch r.Method {
		case "GET":
			jsonResponse(w, 200, a.s.Snapshot().Book)
		case "PUT":
			var b struct {
				Revision  *int64          `json:"revision"`
				Contacts  *[]book.Contact `json:"contacts"`
				UpdatedAt json.RawMessage `json:"updatedAt"`
			}
			if !decode(w, r, &b, a.c.Limits.JSONBytes) {
				return
			}
			if b.Revision == nil || b.Contacts == nil {
				fail(w, 400, fmt.Errorf("נדרשים revision ו־contacts שאינם null"))
				return
			}
			v, e := a.s.Save(*b.Revision, *b.Contacts)
			a.result(w, v, e)
		default:
			w.WriteHeader(405)
		}
	case "/api/info":
		if r.Method != "GET" {
			w.WriteHeader(405)
			return
		}
		v := a.s.Snapshot()
		jsonResponse(w, 200, map[string]any{"urls": a.urls(), "revision": v.Book.Revision, "updatedAt": v.Book.UpdatedAt, "degraded": v.Degraded, "format": a.c.Format, "limits": a.c.Limits})
	case "/api/import/preview":
		if r.Method != "POST" {
			w.WriteHeader(405)
			return
		}
		var body struct {
			Data []byte `json:"data"`
		}
		if !decode(w, r, &body, a.c.Limits.CSVBytes*4/3+4096) {
			return
		}
		p, e := book.ImportCSV(body.Data, a.c.Limits)
		if e != nil {
			fail(w, 400, e)
			return
		}
		jsonResponse(w, 200, p)
	case "/api/export.csv":
		if r.Method != "GET" {
			w.WriteHeader(405)
			return
		}
		b, e := book.ExportCSV(a.s.Snapshot().Book.Contacts)
		if e != nil {
			fail(w, 500, e)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="nimbus-phonebook.csv"`)
		w.Write(b)
	case "/api/backups":
		if r.Method != "GET" {
			w.WriteHeader(405)
			return
		}
		b, e := a.s.Backups()
		if e != nil {
			fail(w, 500, e)
			return
		}
		jsonResponse(w, 200, b)
	case "/api/restore":
		if r.Method != "POST" {
			w.WriteHeader(405)
			return
		}
		var b struct {
			ID       string `json:"id"`
			Revision int64  `json:"revision"`
		}
		if !decode(w, r, &b, a.c.Limits.JSONBytes) {
			return
		}
		v, e := a.s.Restore(b.ID, b.Revision)
		a.result(w, v, e)
	case "/":
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(405)
			return
		}
		b, _ := web.Files.ReadFile("index.html")
		b = bytes.ReplaceAll(b, []byte("__TOKEN__"), []byte(a.Token))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Method == "GET" {
			w.Write(b)
		}
	case "/app.js", "/app.css":
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(405)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		b, e := web.Files.ReadFile(name)
		if e != nil {
			http.NotFound(w, r)
			return
		}
		if name == "app.js" {
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		} else {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		}
		if r.Method == "GET" {
			w.Write(b)
		}
	default:
		http.NotFound(w, r)
	}
}
func (a *App) phones(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/phonebook.xml" {
		http.NotFound(w, r)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(405)
		return
	}
	if a.subnet != nil {
		host, _, e := net.SplitHostPort(r.RemoteAddr)
		if e != nil || !a.subnet.Contains(net.ParseIP(host)) {
			w.WriteHeader(403)
			return
		}
	}
	v := a.s.Snapshot()
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("ETag", v.ETag)
	if !v.Book.UpdatedAt.IsZero() {
		w.Header().Set("Last-Modified", v.Book.UpdatedAt.Format(http.TimeFormat))
	}
	if match := r.Header.Get("If-None-Match"); match != "" {
		for _, tag := range strings.Split(match, ",") {
			tag = strings.TrimSpace(tag)
			if tag == "*" || strings.TrimPrefix(tag, "W/") == v.ETag {
				w.WriteHeader(304)
				return
			}
		}
	} else if t, e := http.ParseTime(r.Header.Get("If-Modified-Since")); e == nil && !v.Book.UpdatedAt.IsZero() && !v.Book.UpdatedAt.Truncate(time.Second).After(t) {
		w.WriteHeader(304)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(v.XML)))
	if r.Method == "GET" {
		w.Write(v.XML)
	}
}
func (a *App) urls() []string {
	out := []string{}
	ifs, _ := net.InterfaceAddrs()
	for _, addr := range ifs {
		ip, _, e := net.ParseCIDR(addr.String())
		if e == nil && ip.To4() != nil && !ip.IsLoopback() {
			if a.subnet == nil || a.subnet.Contains(ip) {
				out = append(out, fmt.Sprintf("http://%s:%d/phonebook.xml", ip, a.c.PhonePort))
			}
		}
	}
	return out
}
func (a *App) Run(ctx context.Context) error { return a.RunReady(ctx, func() {}) }
func (a *App) RunReady(ctx context.Context, ready func()) error {
	admin, e := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", a.c.AdminPort))
	if e != nil {
		return e
	}
	phones, e := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", a.c.PhonePort))
	if e != nil {
		admin.Close()
		return e
	}
	defer admin.Close()
	defer phones.Close()
	makeServer := func(h http.Handler) *http.Server {
		return &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	}
	as, ps := makeServer(a.Admin), makeServer(a.Phones)
	errs := make(chan error, 2)
	go func() { errs <- as.Serve(admin) }()
	go func() { errs <- ps.Serve(phones) }()
	log.Printf("admin http://127.0.0.1:%d; phones port %d", a.c.AdminPort, a.c.PhonePort)
	ready()
	select {
	case <-ctx.Done():
		e = nil
	case e = <-errs:
	}
	stop, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ae := as.Shutdown(stop)
	pe := ps.Shutdown(stop)
	if e != nil && !errors.Is(e, http.ErrServerClosed) {
		return e
	}
	return errors.Join(ae, pe)
}
