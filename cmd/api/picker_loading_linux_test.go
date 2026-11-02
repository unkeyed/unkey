package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tonistiigi/vt100"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
	"golang.org/x/sys/unix"
)

func TestPickerLoadingDiscardsTypeaheadBeforeDestructiveSelection(t *testing.T) {
	for _, reload := range []string{"initial", "page", "search"} {
		t.Run(reload, func(t *testing.T) {
			firstID, selectedID := uid.New(uid.ProjectPrefix), uid.New(uid.ProjectPrefix)
			started, release := make(chan struct{}), make(chan struct{})
			mutations := make(chan components.V2ProjectsDeleteProjectRequestBody, 2)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/projects.listProjects":
					var request components.V2ProjectsListProjectsRequestBody
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						return
					}
					blocked := reload == "initial" || request.Cursor != nil || request.Search != nil
					id := firstID
					if blocked {
						close(started)
						select {
						case <-release:
						case <-r.Context().Done():
							return
						}
						id = selectedID
					}
					response := components.V2ProjectsListProjectsResponseBody{
						Data:       []components.Project{{ID: id, Name: strings.Repeat("Similar project ", 5)}},
						Pagination: components.Pagination{HasMore: !blocked, Cursor: &firstID},
					}
					if err := json.NewEncoder(w).Encode(response); err != nil {
						t.Error(err)
					}
				case "/v2/projects.deleteProject":
					var request components.V2ProjectsDeleteProjectRequestBody
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						return
					}
					mutations <- request
					w.WriteHeader(http.StatusAccepted)
					if err := json.NewEncoder(w).Encode(components.V2ProjectsDeleteProjectResponseBody{}); err != nil {
						t.Error(err)
					}
				default:
					t.Errorf("unexpected endpoint %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			t.Cleanup(server.Close)
			p := startPickerPTY(t, []string{"projects", "delete", "--yes", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL}, 40)
			if reload != "initial" {
				p.expect(firstID)
				if reload == "page" {
					p.send("\x1b[B\r\r\r")
				} else {
					p.send("needle\x1b[B\r\r\r")
				}
			}
			p.expect("Loading resources")
			select {
			case <-started:
			case <-p.ctx.Done():
				t.Fatal("list request did not start")
			}
			state, err := unix.IoctlGetTermios(int(p.master.Fd()), unix.TCGETS)
			require.NoError(t, err)
			require.Zero(t, state.Lflag&(unix.ECHO|unix.ICANON))
			p.send("UNSEEN-TYPEAHEAD\r\r")
			close(release)
			p.expect(selectedID)
			p.observe(150 * time.Millisecond)
			screen := p.screen()
			require.Contains(t, screen, "? Project")
			require.Contains(t, screen, selectedID)
			require.Contains(t, screen, "Ctrl+C to cancel")
			require.NotContains(t, screen, "Delete project")
			require.NotContains(t, p.transcript.String(), "UNSEEN-TYPEAHEAD")
			require.Empty(t, mutations)
			p.send("\r")
			p.expect("Delete project")
			require.Empty(t, mutations)
			p.send("\x1b[H\r")
			require.NoError(t, p.finish(), p.transcript.String())
			require.Len(t, mutations, 1)
			require.Equal(t, selectedID, (<-mutations).Project)
		})
	}
}

func TestPickerLoadingCancellationCancelsPendingRequest(t *testing.T) {
	started, cancelled := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/projects.listProjects" {
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			return
		}
		var request components.V2ProjectsListProjectsRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		close(started)
		select {
		case <-r.Context().Done():
			close(cancelled)
		case <-t.Context().Done():
		}
	}))
	t.Cleanup(server.Close)
	p := startPickerPTY(t, []string{"projects", "delete", "--yes", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL}, 80)
	p.expect("Loading resources")
	select {
	case <-started:
	case <-p.ctx.Done():
		t.Fatal("list request did not start")
	}
	p.send("\x03")
	require.Error(t, p.finish())
	require.Contains(t, p.transcript.String(), "cancelled")
	select {
	case <-cancelled:
	case <-p.ctx.Done():
		t.Fatal("request was not cancelled")
	}
}

func TestPickerNormalLoadAndDeniedListManualEntry(t *testing.T) {
	for _, denied := range []bool{false, true} {
		t.Run(strconv.FormatBool(denied), func(t *testing.T) {
			id := uid.New(uid.ProjectPrefix)
			requests := make(chan components.V2ProjectsGetProjectRequestBody, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/projects.listProjects":
					if denied {
						w.WriteHeader(http.StatusForbidden)
						return
					}
					if err := json.NewEncoder(w).Encode(components.V2ProjectsListProjectsResponseBody{Data: []components.Project{{ID: id, Name: "Normal project"}}}); err != nil {
						t.Error(err)
					}
				case "/v2/projects.getProject":
					var request components.V2ProjectsGetProjectRequestBody
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						return
					}
					requests <- request
					if err := json.NewEncoder(w).Encode(components.V2ProjectsGetProjectResponseBody{Data: components.Project{ID: id, Name: "Normal project"}}); err != nil {
						t.Error(err)
					}
				default:
					t.Errorf("unexpected endpoint %s", r.URL.Path)
				}
			}))
			t.Cleanup(server.Close)
			p := startPickerPTY(t, []string{"projects", "get", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL}, 80)
			if denied {
				p.expect("Enter the project ID instead")
				p.send(id + "\r")
			} else {
				p.expect(id)
				p.send("\r")
			}
			require.NoError(t, p.finish(), p.transcript.String())
			require.Len(t, requests, 1)
			require.Equal(t, id, (<-requests).Project)
		})
	}
}

func TestPickerEscapeClearsQueryPreservesTextAndSplitArrows(t *testing.T) {
	firstID, secondID := uid.New(uid.ProjectPrefix), uid.New(uid.ProjectPrefix)
	requests := make(chan components.V2ProjectsGetProjectRequestBody, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/projects.listProjects":
			if err := json.NewEncoder(w).Encode(components.V2ProjectsListProjectsResponseBody{Data: []components.Project{{ID: firstID, Name: "Alpha"}, {ID: secondID, Name: "Beta"}}}); err != nil {
				t.Error(err)
			}
		case "/v2/projects.getProject":
			var request components.V2ProjectsGetProjectRequestBody
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
				return
			}
			requests <- request
			if err := json.NewEncoder(w).Encode(components.V2ProjectsGetProjectResponseBody{Data: components.Project{ID: request.Project, Name: "Beta"}}); err != nil {
				t.Error(err)
			}
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	args := []string{"projects", "get", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL}
	p := startPickerPTY(t, args, 80)
	p.expect(secondID)
	p.send("zz")
	p.expect("Use: zz")
	p.send("\x1b")
	p.expect("Filter results or enter a value")
	p.send("bet")
	p.expect("Use: bet")
	p.send("\x1bA")
	p.expect("Use: A")
	p.send("\x1b")
	p.expect("Filter results or enter a value")
	p.send("\x1b")
	p.observe(10 * time.Millisecond)
	p.send("[")
	p.observe(10 * time.Millisecond)
	p.send("B")
	p.expect("❯ Beta")
	p.send("\r")
	require.NoError(t, p.finish(), p.transcript.String())
	require.Len(t, requests, 1)
	require.Equal(t, secondID, (<-requests).Project)

	p = startPickerPTY(t, args, 80)
	p.expect(secondID)
	p.send("\x1b")
	require.Error(t, p.finish())
	require.Contains(t, p.transcript.String(), "cancelled")
	require.Empty(t, requests)
}

func TestPickerParentChoiceHasEntryAndCancelHints(t *testing.T) {
	p := startPickerPTY(t, []string{"keys", "get"}, 80)
	p.expect("Choose API ID to browse")
	require.Contains(t, p.screen(), "Enter a value")
	require.Contains(t, p.screen(), "Ctrl+C to cancel")
	require.NotContains(t, p.screen(), "filter")
	p.send("\x1b")
	require.Error(t, p.finish())
	require.Contains(t, p.transcript.String(), "cancelled")
}

type pickerPTY struct {
	t          *testing.T
	ctx        context.Context
	master     *os.File
	chunks     chan string
	done       chan struct{}
	waitErr    error
	transcript strings.Builder
	position   int
	columns    int
}

func startPickerPTY(t *testing.T, args []string, columns uint16) *pickerPTY {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, master.Close()) })
	require.NoError(t, unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0))
	number, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	require.NoError(t, err)
	slave, err := os.OpenFile("/dev/pts/"+strconv.Itoa(number), os.O_RDWR|unix.O_NOCTTY, 0)
	require.NoError(t, err)
	require.NoError(t, unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Col: columns, Row: 24}))
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	raw, err := json.Marshal(args)
	require.NoError(t, err)
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGuidedProcess$")
	command.Env = append(os.Environ(), "UNKEY_GUIDED_PROCESS=1", "UNKEY_GUIDED_ARGS="+string(raw), "UNKEY_OUTPUT=human", "TERM=xterm-256color", "NO_COLOR=1")
	command.Stdin, command.Stderr = slave, slave
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	var output bytes.Buffer
	command.Stdout = &output
	p := &pickerPTY{t: t, ctx: ctx, master: master, chunks: make(chan string, 256), done: make(chan struct{}), columns: int(columns)}
	require.NoError(t, command.Start())
	require.NoError(t, slave.Close())
	go func() {
		p.waitErr = command.Wait()
		close(p.done)
	}()
	t.Cleanup(func() {
		cancel()
		<-p.done
	})
	go func() {
		defer close(p.chunks)
		buffer := make([]byte, 4096)
		for {
			n, err := master.Read(buffer)
			if n > 0 {
				select {
				case p.chunks <- string(buffer[:n]):
				case <-ctx.Done():
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	return p
}

func (p *pickerPTY) send(input string) {
	p.t.Helper()
	_, err := io.WriteString(p.master, input)
	require.NoError(p.t, err)
}

func (p *pickerPTY) expect(text string) {
	p.t.Helper()
	for !strings.Contains(p.transcript.String()[p.position:], text) {
		select {
		case chunk, ok := <-p.chunks:
			require.True(p.t, ok, "process ended before %q: %s", text, p.transcript.String())
			p.transcript.WriteString(chunk)
		case <-p.ctx.Done():
			p.t.Fatalf("waiting for %q: %s", text, p.transcript.String())
		}
	}
	p.position = p.transcript.Len()
}

func (p *pickerPTY) observe(duration time.Duration) {
	p.t.Helper()
	timer := time.NewTimer(duration)
	defer timer.Stop()
	for {
		select {
		case chunk, ok := <-p.chunks:
			require.True(p.t, ok, "process ended without a fresh choice: %s", p.transcript.String())
			p.transcript.WriteString(chunk)
		case <-timer.C:
			return
		case <-p.ctx.Done():
			p.t.Fatal("process timed out")
		}
	}
}

func (p *pickerPTY) screen() string {
	p.t.Helper()
	screen := vt100.NewVT100(24, p.columns)
	_, err := screen.Write([]byte(p.transcript.String()))
	require.NoError(p.t, err)
	var visible strings.Builder
	for _, row := range screen.Content {
		visible.WriteString(strings.TrimRight(string(row), " ") + "\n")
	}
	return visible.String()
}

func (p *pickerPTY) finish() error {
	p.t.Helper()
	<-p.done
	for chunk := range p.chunks {
		p.transcript.WriteString(chunk)
	}
	return p.waitErr
}
