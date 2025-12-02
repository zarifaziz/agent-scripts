package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"sanity-web/core"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type hub struct {
	clients   map[*websocket.Conn]bool
	broadcast chan []byte
	history   [][]byte
	mu        sync.Mutex
}

func newHub() *hub {
	return &hub{clients: map[*websocket.Conn]bool{}, broadcast: make(chan []byte, 1024)}
}

func (h *hub) run() {
	for msg := range h.broadcast {
		h.mu.Lock()
		h.history = append(h.history, msg)
		for c := range h.clients {
			if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.Close()
				delete(h.clients, c)
			}
		}
		h.mu.Unlock()
	}
}

func (h *hub) addClient(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	h.mu.Lock()
	h.clients[conn] = true
	// replay history to new client
	for _, msg := range h.history {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			conn.Close()
			delete(h.clients, conn)
			h.mu.Unlock()
			return
		}
	}
	h.mu.Unlock()
}

func (h *hub) handleStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, msg := range h.history {
		w.Write(msg)
		if len(msg) == 0 || msg[len(msg)-1] != '\n' {
			w.Write([]byte("\n"))
		}
	}
}

func (h *hub) handleReset(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.history = nil
	h.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (h *hub) handlePinSave(w http.ResponseWriter, r *http.Request) {
	// read body as NDJSON and persist to cache dir
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read", http.StatusBadRequest)
		return
	}
	if len(data) == 0 {
		http.Error(w, "empty", http.StatusBadRequest)
		return
	}
	id := fmt.Sprintf("%x", time.Now().UnixNano())
	if err := savePin(id, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("pin saved id=%s bytes=%d", id, len(data))
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id":"%s","url":"/p/%s"}`, id, id)
}

func (h *hub) handlePinLoad(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/pin/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	data, err := loadPin(id)
	if err != nil {
		log.Printf("pin load failed id=%s: %v", id, err)
		http.NotFound(w, r)
		return
	}
	log.Printf("pin load id=%s bytes=%d", id, len(data))
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Write(data)
}

func (h *hub) handlePinDelete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/pin-delete/"))
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if err := deletePin(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("pin delete id=%s", id)
	w.WriteHeader(http.StatusOK)
}

func main() {
	file := flag.String("file", "", "log file (defaults to stdin)")
	fx := flag.Bool("fx", false, "include [Fx] logs")
	minLines := flag.Int("min-lines", 4, "min lines before block extraction")
	cfgPath := flag.String("config", "", "config file override")
	addr := flag.String("addr", ":5151", "listen address")
	noTee := flag.Bool("no-tee", false, "disable tee to sane CLI")
	flag.Parse()

	// ensure older instances don't hold the port
	killExisting(*addr)

	cfg, err := core.LoadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	in := inputReader(*file)
	defer in.Close()

	h := newHub()
	go h.run()

	// Determine if we should tee to sane (when reading from pipe, not file)
	shouldTee := *file == "" && !*noTee

	go func() {
		var reader io.Reader = in
		var teeWriters []io.Writer

		// Always tee raw input to log file
		logFile, logPath := openRawLogFile()
		if logFile != nil {
			teeWriters = append(teeWriters, logFile)
			defer logFile.Close()
			log.Printf("raw log: %s", logPath)
		}

		// If piped input, also tee to sane for CLI output
		var saneCmd *exec.Cmd
		var sanePipe io.WriteCloser
		if shouldTee {
			saneCmd = exec.Command("sane")
			saneCmd.Stdout = os.Stdout
			saneCmd.Stderr = os.Stderr

			var err error
			sanePipe, err = saneCmd.StdinPipe()
			if err != nil {
				log.Printf("sane pipe: %v", err)
			} else {
				if err := saneCmd.Start(); err != nil {
					log.Printf("sane start: %v", err)
					sanePipe = nil
				} else {
					teeWriters = append(teeWriters, sanePipe)
					log.Printf("tee enabled: piping to sane CLI")
				}
			}
		}

		// Set up tee if we have any writers
		if len(teeWriters) > 0 {
			reader = io.TeeReader(in, io.MultiWriter(teeWriters...))
		}

		p := core.NewProcessor(*cfg, core.Options{IncludeFXLogs: *fx, MinMultilineLines: *minLines})
		if err := p.ProcessStream(bufio.NewReader(reader), broadcastWriter{hub: h}); err != nil {
			log.Printf("process: %v", err)
		}

		// Close sane stdin pipe and wait for it to finish after processing completes
		if sanePipe != nil {
			sanePipe.Close()
		}
		if saneCmd != nil && saneCmd.Process != nil {
			saneCmd.Wait()
		}

		h.mu.Lock()
		log.Printf("stream complete, history records: %d", len(h.history))
		h.mu.Unlock()
	}()

	webDir, err := findWebDir()
	if err != nil {
		log.Fatalf("locate web assets: %v", err)
	}
	http.HandleFunc("/ws", h.addClient)
	http.HandleFunc("/stream", h.handleStream)
	http.HandleFunc("/reset", h.handleReset)
	http.HandleFunc("/pin", h.handlePinSave)
	http.HandleFunc("/pin/", h.handlePinLoad)
	http.HandleFunc("/pin-delete/", h.handlePinDelete)
	http.HandleFunc("/p/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
	})
	fs := http.FileServer(http.Dir(webDir))
	http.Handle("/", fs)

	log.Printf("listening on %s", *addr)
	log.Printf("open http://localhost%v", *addr)
	if os.Getenv("OPEN_BROWSER") != "0" {
		go openBrowser("http://localhost" + *addr)
	}
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}

type broadcastWriter struct{ hub *hub }

func (b broadcastWriter) Write(p []byte) (int, error) {
	if len(p) > 0 {
		b.hub.broadcast <- append([]byte{}, p...)
	}
	return len(p), nil
}

func inputReader(path string) *os.File {
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			log.Fatalf("open file: %v", err)
		}
		return f
	}
	st, _ := os.Stdin.Stat()
	if (st.Mode() & os.ModeCharDevice) != 0 {
		log.Fatalf("provide -file or pipe logs")
	}
	return os.Stdin
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	}
	if err != nil {
		log.Printf("browser open failed: %v", err)
	}
}

func killExisting(addr string) {
	// best-effort: kill other sanityweb processes to free port
	_ = exec.Command("pkill", "-f", "sanityweb").Run()
	// also clear any process occupying the target port
	port := extractPort(addr)
	if port != "" {
		out, err := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%s", port)).Output()
		if err == nil && len(out) > 0 {
			pids := strings.Fields(string(out))
			for _, pid := range pids {
				_ = exec.Command("kill", pid).Run()
			}
		}
	}
}

func extractPort(addr string) string {
	// addr formats: :8080, 0.0.0.0:8080, localhost:8080
	if addr == "" {
		return ""
	}
	parts := strings.Split(addr, ":")
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[len(parts)-1]
}

func findWebDir() (string, error) {
	// try resolving from executable dir, then ascend
	if exe, err := os.Executable(); err == nil {
		if real, err2 := filepath.EvalSymlinks(exe); err2 == nil {
			exe = real
		}
		if dir := searchUpForWeb(filepath.Dir(exe)); dir != "" {
			return dir, nil
		}
	}
	// try from cwd
	if cwd, err := os.Getwd(); err == nil {
		if dir := searchUpForWeb(cwd); dir != "" {
			return dir, nil
		}
	}
	return "", fmt.Errorf("web directory not found near executable or CWD")
}

func searchUpForWeb(start string) string {
	dir := start
	for {
		candidate := filepath.Join(dir, "web")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			// basic sanity: needs index.html
			if _, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil {
				return candidate
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir { // root
			break
		}
		dir = parent
	}
	return ""
}

func pinDir() (string, error) {
	base, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, ".cache", "scripts", "sanity-web")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func savePin(id string, data []byte) error {
	dir, err := pinDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, id+".ndjson")
	return os.WriteFile(path, data, 0o644)
}

func loadPin(id string) ([]byte, error) {
	dir, err := pinDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, id+".ndjson")
	// allow id with .ndjson suffix
	if _, err := os.Stat(path); os.IsNotExist(err) && !strings.HasSuffix(path, ".ndjson") {
		path = filepath.Join(dir, id)
	}
	return os.ReadFile(path)
}

func deletePin(id string) error {
	dir, err := pinDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, id+".ndjson")
	if _, err := os.Stat(path); os.IsNotExist(err) && !strings.HasSuffix(path, ".ndjson") {
		path = filepath.Join(dir, id)
	}
	return os.Remove(path)
}

func openRawLogFile() (*os.File, string) {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("raw log: home dir: %v", err)
		return nil, ""
	}
	dir := filepath.Join(home, ".cache", "scripts", "sanity-web")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("raw log: mkdir: %v", err)
		return nil, ""
	}
	filename := time.Now().Format("2006-01-02_15-04-05") + ".log"
	path := filepath.Join(dir, filename)
	f, err := os.Create(path)
	if err != nil {
		log.Printf("raw log: create: %v", err)
		return nil, ""
	}
	return f, path
}
