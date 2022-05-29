package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {

	ctx, cancel := gracefulCtx()
	defer cancel()

	s, err := newStorage(ctx)
	if err != nil {
		panic(err)
	}
	h, err := newHandler(s)
	if err != nil {
		panic(err)
	}

	r := mux.NewRouter()
	r.PathPrefix("/").Methods(http.MethodGet).HandlerFunc(h.handleGet)
	r.PathPrefix("/").Methods(http.MethodPost).HandlerFunc(h.handlePost)
	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf(":%d", 1999),
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println(err)
		}
	}()
	fmt.Println("server listening")

	<-ctx.Done()
	fmt.Println("context done")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Println(err)
	}

	wg.Wait()
	fmt.Println("successful shutdown")
}

//go:embed index.html.tpl
var indexHtml string

type handler struct {
	s    *storage
	tpl  *template.Template
	pool *sync.Pool
}

func newHandler(s *storage) (*handler, error) {
	tpl, err := template.New("index.html").Parse(indexHtml)
	if err != nil {
		return nil, err
	}
	return &handler{
		s:    s,
		tpl:  tpl,
		pool: new(sync.Pool),
	}, nil
}

// FIXME: mpavlicek - move parsing and marshaling to the update part of this script
func (h *handler) handleGet(w http.ResponseWriter, r *http.Request) {
	jsonTimeline, err := json.Marshal(h.s.timeline)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "marshal error: %v", err)
		return
	}
	buf := new(bytes.Buffer)
	if err := h.tpl.ExecuteTemplate(buf, "index.html", struct {
		Timeline string
	}{
		Timeline: string(jsonTimeline),
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "execute template error: %v", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	io.Copy(w, buf)
}

func (h *handler) handlePost(w http.ResponseWriter, r *http.Request) {
	if err := h.s.update(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "fetch error: %v", err)
		return
	}
	h.handleGet(w, r)
}

type storage struct {
	timeline Timeline
	mut      *sync.Mutex
}

func newStorage(ctx context.Context) (*storage, error) {
	t, err := fetch(ctx)
	if err != nil {
		return nil, err
	}
	return &storage{
		timeline: t,
		mut:      new(sync.Mutex),
	}, nil
}

func (r *storage) update(ctx context.Context) error {
	t, err := fetch(ctx)
	if err != nil {
		return err
	}
	r.mut.Lock()
	defer r.mut.Unlock()
	r.timeline = t
	return nil
}

func (r *storage) get() Timeline {
	r.mut.Lock()
	defer r.mut.Unlock()
	return r.timeline
}

const (
	fieldStart = iota
	fieldEnd
	fieldTitle
	fieldDescription
	fieldGroup
)

func fetch(ctx context.Context) (Timeline, error) {

	var creds *google.Credentials
	if raw, has := os.LookupEnv("GOOGLE_JSON_CREDENTIALS"); has {
		var err error
		creds, err = google.CredentialsFromJSON(ctx, []byte(raw), sheets.SpreadsheetsReadonlyScope)
		if err != nil {
			return Timeline{}, fmt.Errorf("error getting credentials: %v", err)
		}
	} else {
		return Timeline{}, fmt.Errorf("GOOGLE_JSON_CREDENTIALS not found")
	}

	srv, err := sheets.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return Timeline{}, err
	}

	var t Timeline

	events, err := srv.Spreadsheets.Values.Get(os.Getenv("SPREADSHEET_ID"), "events!A2:E").Do()
	if err != nil {
		return Timeline{}, err
	}
	for i, line := range events.Values {
		lineIndex := i + 2

		var e Event
		if start, ok := line[fieldStart].(string); ok && start != "" {
			startTime, err := time.Parse("2006-1", start)
			if err != nil {
				return Timeline{}, fmt.Errorf("line %d: error parsing start date: %v", lineIndex, err)
			}
			e.Start = Date{
				Year:  startTime.Year(),
				Month: int(startTime.Month()),
			}
		} else {
			return Timeline{}, fmt.Errorf("line %d: start date missing or corrupt", lineIndex)
		}
		if end, ok := line[fieldEnd].(string); ok && end != "" {
			endTime, err := time.Parse("2006-1", end)
			if err != nil {
				return Timeline{}, fmt.Errorf("line %d: error parsing end date: %v", lineIndex, err)
			}
			e.End = &Date{
				Year:  endTime.Year(),
				Month: int(endTime.Month()),
			}
		}
		if title, ok := line[fieldTitle].(string); ok && title != "" {
			e.Text.Title = title
		} else {
			return Timeline{}, fmt.Errorf("line %d: title missing or corrupt", lineIndex)
		}
		if len(line) >= 4 {
			if description, ok := line[fieldDescription].(string); ok && description != "" {
				e.Text.Description = &description
			}
		}
		if len(line) >= 5 {
			if group, ok := line[fieldGroup].(string); ok && group != "" {
				e.Group = &group
			}
		}
		t.Events = append(t.Events, e)
	}

	eras, err := srv.Spreadsheets.Values.Get(os.Getenv("SPREADSHEET_ID"), "eras!A2:C").Do()
	if err != nil {
		return Timeline{}, err
	}
	for i, line := range eras.Values {
		lineIndex := i + 2

		var e Era
		if start, ok := line[fieldStart].(string); ok && start != "" {
			startTime, err := time.Parse("2006-1", start)
			if err != nil {
				return Timeline{}, fmt.Errorf("line %d: error parsing start date: %v", lineIndex, err)
			}
			e.Start = Date{
				Year:  startTime.Year(),
				Month: int(startTime.Month()),
			}
		} else {
			return Timeline{}, fmt.Errorf("line %d: start date missing or corrupt", lineIndex)
		}
		if end, ok := line[fieldEnd].(string); ok && end != "" {
			endTime, err := time.Parse("2006-1", end)
			if err != nil {
				return Timeline{}, fmt.Errorf("line %d: error parsing end date: %v", lineIndex, err)
			}
			e.End = Date{
				Year:  endTime.Year(),
				Month: int(endTime.Month()),
			}
		} else {
			return Timeline{}, fmt.Errorf("line %d: end date missing or corrupt", lineIndex)
		}
		if title, ok := line[fieldTitle].(string); ok && title != "" {
			e.Text.Title = title
		} else {
			return Timeline{}, fmt.Errorf("line %d: title missing or corrupt", lineIndex)
		}
		t.Eras = append(t.Eras, e)
	}

	return t, nil
}

type Timeline struct {
	Events []Event `json:"events"`
	Eras   []Era   `json:"eras"`
}

type Date struct {
	Year  int `json:"year"`
	Month int `json:"month"`
}

type Text struct {
	Title       string  `json:"headline"`       // can be HTML
	Description *string `json:"text,omitempty"` // can be HTML
}

type Event struct {
	Start Date    `json:"start_date"`
	End   *Date   `json:"end_date,omitempty"`
	Text  Text    `json:"text"`
	Group *string `json:"group,omitempty"`
}

type Era struct {
	Start Date `json:"start_date"`
	End   Date `json:"end_date"`
	Text  Text `json:"text"`
}

func gracefulCtx() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	signals := make(chan os.Signal)
	signal.Notify(signals, terminationSignals()...)

	go func() {
		select {
		case <-ctx.Done():
		case <-signals:
			cancel()
		}
	}()

	return ctx, cancel
}

// terminationSignals from https://www.gnu.org/software/libc/manual/html_node/Termination-Signals.html
func terminationSignals() []os.Signal {
	return []os.Signal{
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGKILL,
		syscall.SIGHUP,
	}
}
