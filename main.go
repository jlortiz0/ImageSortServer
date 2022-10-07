package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"
)

const defaultRootDir = "jlortiz_TEST" + string(os.PathSeparator)

var rootDir string = defaultRootDir

func main() {
	// TODO: flags
	loadHashes()
	hndlr := http.NewServeMux()
	// rootDir = defaultRootDir
	hndlr.HandleFunc("/api/1", apiHandler)
	hndlr.Handle("/api", http.NotFoundHandler())
	// TODO: Custom file server that handles renames and bars PUT
	// Also mount www somewhere as a GET only file server
	hndlr.Handle("/", http.FileServer(http.Dir(rootDir)))
	srv := &http.Server{ReadHeaderTimeout: time.Second * 5, Handler: hndlr}
	go srv.ListenAndServe()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
	srv.Shutdown(context.Background())
	saveHashes()
}

func writeAll(w io.Writer, b []byte) error {
	for len(b) > 0 {
		c, err := w.Write(b)
		if err != nil {
			return err
		}
		b = b[c:]
	}
	return nil
}
