package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
    "unsafe"

    prettyjson "github.com/hokaccha/go-prettyjson"
    "golang.org/x/sys/unix"

	"sanity-web/core"
)

type options struct {
	includeFX    bool
	minLines     int
	cfgPath      string
	inputPath    string
	outputNDJSON bool
	noColor      bool
}

func parseFlags() options {
	var opt options
	flag.BoolVar(&opt.includeFX, "fx", false, "include [Fx] framework logs")
	flag.IntVar(&opt.minLines, "min-lines", 4, "minimum lines before a string becomes a block")
	flag.StringVar(&opt.cfgPath, "config", "", "optional config file for key grouping")
	flag.StringVar(&opt.inputPath, "file", "", "log file (defaults to stdin)")
	flag.BoolVar(&opt.outputNDJSON, "ndjson", false, "output NDJSON instead of pretty view")
	flag.BoolVar(&opt.noColor, "no-color", false, "disable ANSI colors")
	flag.Parse()
	return opt
}

func openInput(opt options) io.ReadCloser {
	if opt.inputPath != "" {
		f, err := os.Open(opt.inputPath)
		if err != nil {
			log.Fatalf("open input: %v", err)
		}
		return f
	}
	info, err := os.Stdin.Stat()
	if err != nil {
		log.Fatalf("stat stdin: %v", err)
	}
	if (info.Mode() & os.ModeCharDevice) != 0 {
		fmt.Fprintln(os.Stderr, "usage: cat server.log | sane [flags]  or  sane -file server.log")
		flag.PrintDefaults()
		os.Exit(1)
	}
	return os.Stdin
}

func main() {
	opt := parseFlags()

	cfg, err := core.LoadConfig(opt.cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	in := openInput(opt)
	defer in.Close()

	p := core.NewProcessor(*cfg, core.Options{IncludeFXLogs: opt.includeFX, MinMultilineLines: opt.minLines})

	if opt.outputNDJSON {
		if err := p.ProcessStream(bufio.NewReader(in), os.Stdout); err != nil {
			log.Fatalf("process: %v", err)
		}
		return
	}

	var buf bytes.Buffer
	if err := p.ProcessStream(bufio.NewReader(in), &buf); err != nil {
		log.Fatalf("process: %v", err)
	}

	useColor := shouldColor(opt)
	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	for dec.More() {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			log.Fatalf("decode: %v", err)
		}
		renderRecord(m, useColor)
	}
}

func renderRecord(m map[string]any, color bool) {
	t, _ := m["type"].(string)
	switch t {
	case "text":
		if txt, ok := m["text"].(string); ok {
			fmt.Println(txt)
		}
	case "log":
		if js, ok := m["json"].(map[string]any); ok {
			renderJSON(js, color)
		}
	case "block":
		if blk, ok := m["block"].(map[string]any); ok {
			path, _ := blk["path"].(string)
			val, _ := blk["value"].(string)
			renderBlock(path, val, color)
		}
	default:
		// fallback
		fmt.Println(m)
	}
}

func renderJSON(js map[string]any, color bool) {
    if color {
        f := prettyjson.NewFormatter()
        f.Indent = 2
        out, err := f.Marshal(js)
        if err == nil {
            fmt.Println(string(out))
            return
        }
    }
    pretty, _ := json.MarshalIndent(js, "", "  ")
    fmt.Println(string(pretty))
}

func renderBlock(path, val string, color bool) {
	header := path
	if color {
		header = "\x1b[36m" + path + "\x1b[0m"
	}
	fmt.Printf("%s:\n", header)

	if strings.Contains(strings.ToLower(path), "stacktrace") {
		for _, line := range strings.Split(val, "\n") {
			line = strings.ReplaceAll(line, "\r", "")
			if color {
				switch {
				case strings.HasPrefix(line, "Oops:") || strings.HasPrefix(line, "Thrown:"):
					fmt.Printf("\x1b[1;31m%s\x1b[0m\n", line)
				case strings.HasPrefix(strings.TrimLeft(line, " "), "--- at"):
					fmt.Printf("\x1b[36m%s\x1b[0m\n", line)
				default:
					fmt.Println(line)
				}
			} else {
				fmt.Println(line)
			}
		}
		return
	}

	if color {
		fmt.Printf("\x1b[35m%s\x1b[0m\n", val)
	} else {
		fmt.Println(val)
	}
}

func shouldColor(opt options) bool {
	if opt.noColor || strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
		return false
	}
	fi, _ := os.Stdout.Stat()
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		return false
	}
	// best-effort: check if stdout is a terminal
	var termios unix.Termios
	if _, _, err := unix.Syscall6(unix.SYS_IOCTL, os.Stdout.Fd(), uintptr(unix.TIOCGETA), uintptr(unsafe.Pointer(&termios)), 0, 0, 0); err == 0 {
		return true
	}
	return true
}

// Helper to resolve config relative to invocation directory when embedding default is used.
func defaultConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	candidate := filepath.Join(dir, "fields.json")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}
