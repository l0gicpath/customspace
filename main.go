package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

const maxUploadSize = 24 << 20 // ~25MB

var (
	srv                   *http.Server
	port                  string
	uploadDirPath         string
	acceptableUploadMimes = map[string]struct{}{
		"image/x-icon": {},
		"image/gif":    {},
		"image/png":    {},
		"image/jpeg":   {},
		"image/bmp":    {},
		"image/webp":   {},
	}
)

func serve(ctx context.Context) error {
	var err error
	go func() {
		if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %s\n", err)
		}
	}()
	log.Printf("Server started on port %s\n", port)

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Server shutting down")
	if err = srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Failed to shutdown server: %s\n", err)
	}

	if err == http.ErrServerClosed {
		err = nil
	}

	return err
}

func postImages(rw http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		log.Println("/images endpoint only supports POST requests")

		rw.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintln(rw, "/images endpoint only supports POST requests")
		return
	}
	r.ParseMultipartForm(maxUploadSize)
	file, fh, err := r.FormFile("image")
	if err != nil {
		log.Printf("error handling file upload: %s\n", err)

		rw.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintln(rw, "Error handling file upload")
		return
	}
	defer file.Close()

	b := make([]byte, 512)
	_, err = file.Read(b)
	if err != nil {
		log.Printf("error reading file to determine content type: %s\n", err)

		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(rw, "Something went wrong, please try again")
		return
	}
	filetype := http.DetectContentType(b)
	if _, found := acceptableUploadMimes[filetype]; !found {
		log.Println("user uploaded an unacceptable file format")

		rw.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(rw, "Error handling file upload, please only upload images")
		return
	}

	dstFilename := fmt.Sprintf("./uploads/%s", fh.Filename)
	f, err := os.OpenFile(dstFilename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("error storing uploaded file: %s\n", err)

		rw.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintln(rw, "Error storing uploaded file, please try again")
		return
	}

	bb := bytes.NewReader(b)
	io.Copy(f, bb)
	io.Copy(f, file)
	rw.WriteHeader(http.StatusCreated)
	fmt.Fprintln(rw, "Upload successful")
}

func main() {
	flag.StringVar(&port, "port", "8080", "port for the server to listen on")
	flag.StringVar(&uploadDirPath, "uploaddir", "./uploads", "set the uploads path")
	flag.Parse()

	var err error
	_, err = os.Stat(uploadDirPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(uploadDirPath, 0700); err != nil {
				log.Fatalf("failed to initialize uploads directory: %s\n", err)
			}
		} else {
			log.Fatal(err)
		}
	}

	m := http.NewServeMux()
	fs := http.FileServer(http.Dir("./uploads"))
	m.Handle("/", fs)
	m.HandleFunc("/images", postImages)
	srv = &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           m,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       0, // Use ReadTimeout instead
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ch
		cancel()
	}()
	serve(ctx)
	log.Println("bye.")
}
