// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: GPL-3.0-only

package mdedit

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// Exec executes the mdedit applet and returns an exit code (0 for success, >0 for error).
func Exec(args []string) int {
	if len(args) != 2 {
		os.Stderr.Write([]byte("  Usage: mdedit <file.md> <listenaddr>\nExample: mdedit todolist.md localhost:8080\n"))
		return 1
	}
	markdownPath, listenAddress := args[0], args[1]

	// prepare HTTP server
	l := logic{markdownPath}
	m := http.NewServeMux()
	m.HandleFunc("GET /{$}", l.handleGetFile("index.html", embeddedHTML))
	m.HandleFunc("GET /res.css", l.handleGetFile("res.css", embeddedCSS))
	m.HandleFunc("GET /res.js", l.handleGetFile("res.js", embeddedJS))
	m.HandleFunc("GET /data.html", l.handleGetDataHTML)
	m.HandleFunc("GET /data.md", l.handleGetDataMarkdown)
	m.HandleFunc("PUT /data.md", l.handlePutDataMarkdown)
	s := &http.Server{Addr: listenAddress, Handler: m}

	exitCode := &atomic.Int32{}
	exitCode.Store(0)

	// setup termination of HTTP server on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var wg sync.WaitGroup
	wg.Go(func() {
		for range ctx.Done() {
		}
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		err := s.Shutdown(ctx)
		if err != nil {
			log.Println("shutdown error: " + err.Error())
			exitCode.Store(1)
		}
		cancel()
	})

	// run HTTP server
	log.Printf("listening on %s...\n", listenAddress)
	err := s.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Println("listen error: " + err.Error())
		exitCode.Store(1)
	}
	stop()
	wg.Wait()

	return int(exitCode.Load())
}

type logic struct {
	MarkdownPath string
}

var (
	//go:embed res/index.html
	embeddedHTML []byte
	//go:embed res/style.css
	embeddedCSS []byte
	//go:embed res/app.js
	embeddedJS []byte
)

// Handles `GET /` and `GET /res/...`.
func (l logic) handleGetFile(name string, contents []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(contents))
	}
}

// Handles `GET /data.html`.
func (l logic) handleGetDataHTML(w http.ResponseWriter, r *http.Request) {
	buf, err := os.ReadFile(l.MarkdownPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	l.respondByRenderingMarkdown(w, buf)
}

// Handles `GET /data.md`.
func (l logic) handleGetDataMarkdown(w http.ResponseWriter, r *http.Request) {
	buf, err := os.ReadFile(l.MarkdownPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Mdedit-Path", l.MarkdownPath)
	http.ServeContent(w, r, "data.md", time.Time{}, bytes.NewReader(buf))
}

// Handles `PUT /data.md`.
func (l logic) handlePutDataMarkdown(w http.ResponseWriter, r *http.Request) {
	const maxSizeBytes = 8 * 1024 * 1024 // 8 MiB ought to be enough for a single Markdown file
	buf, err := io.ReadAll(io.LimitReader(r.Body, maxSizeBytes))
	if err != nil {
		http.Error(w, "while reading request body: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = os.WriteFile(l.MarkdownPath, buf, 0666)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	l.respondByRenderingMarkdown(w, buf)
}

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

func (l logic) respondByRenderingMarkdown(w http.ResponseWriter, source []byte) {
	var buf bytes.Buffer
	err := md.Convert(source, &buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}
