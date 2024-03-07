package main

import (
	"bufio"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lmittmann/tint"
)

type Content struct {
	Data         map[string]string
	mu           sync.Mutex
	CleanupTimer *time.Timer
	CleanupAfter time.Duration
}

func (content *Content) load(path string) {
	results := map[string]string{}
	resultsMutex := sync.Mutex{}

	wg := sync.WaitGroup{}
	seats := make(chan struct{}, 4)

	file, _ := os.Open(path)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		wg.Add(1)
		seats <- struct{}{} // acquire "seat"

		go func(line string) {
			defer wg.Done()
			defer func() { <-seats }() // release "seat"

			result := strings.Split(line, ";")
			if len(result) != 2 {
				return
			}
			if !strings.HasPrefix(result[1], "https://") {
				return
			}
			resultsMutex.Lock()
			results[result[0]] = result[1]
			resultsMutex.Unlock()
		}(scanner.Text())
	}

	wg.Wait()
	content.mu.Lock()
	defer content.mu.Unlock()
	content.Data = results
}

func (content *Content) clear() {
	if content.Data == nil {
		return
	}
	content.mu.Lock()
	defer content.mu.Unlock()
	content.Data = nil
}

func (content *Content) NukeDataInMemAfterDuration() {
	content.CleanupTimer = time.NewTimer(0)
	content.CleanupAfter = 5 * time.Minute

	cleanupAfterEnv := os.Getenv("CLEANUP_AFTER")
	if cleanupAfterEnv != "" {
		duration, err := time.ParseDuration(cleanupAfterEnv)
		if err == nil {
			content.CleanupAfter = duration
		}
	}

	for {
		<-content.CleanupTimer.C
		content.clear()
		slog.Debug("Memory freed up")
	}
}

func (content *Content) GetLink(path string) (string, bool) {
	content.CleanupTimer.Stop()
	content.CleanupTimer.Reset(content.CleanupAfter)
	if content.Data == nil {
		content.load("routes.csv")
		slog.Debug("Loaded routes.csv")
	}
	link, ok := content.Data[path]
	return link, ok
}

func main() {
	logLevel := slog.LevelInfo
	if os.Getenv("DEBUG") == "true" {
		logLevel = slog.LevelDebug
	}

	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      logLevel,
			TimeFormat: time.RFC1123Z,
		}),
	))

	content := &Content{}
	go content.NukeDataInMemAfterDuration()

	http.HandleFunc("GET /{path}", func(w http.ResponseWriter, r *http.Request) {
		pathString := r.PathValue("path")
		if pathString == "" {
			w.Header().Set("Content-Type", "text/plain")
			http.Error(w, "Path empty", http.StatusNotFound)
			return
		}
		if link, ok := content.GetLink(pathString); ok {
			http.Redirect(w, r, link, http.StatusFound)
			return
		}
		http.Error(w, "Path not found", http.StatusNotFound)
	})

	slog.Info("Server started", "port", 8080)
	http.ListenAndServe(":8080", nil)
}
