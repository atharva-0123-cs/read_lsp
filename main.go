package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/atharva-0123-cs/read_lsp/server"
)

const version = "0.4.0"

func main() {
	logPath := flag.String("log", "./log.txt", "path to the server log file")
	showVersion := flag.Bool("version", false, "print the server version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("read_lsp", version)
		return
	}

	logger := getLogger(*logPath)
	logger.Printf("read_lsp %s starting", version)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		_ = os.Stdin.Close()
	}()

	srv := server.New(server.Options{Logger: logger})
	if err := srv.Serve(ctx, os.Stdin, os.Stdout); err != nil && ctx.Err() == nil {
		logger.Printf("server stopped with error: %s", err)
		os.Exit(1)
	}

	logger.Print("server stopped")
}

func getLogger(filename string) *log.Logger {
	logfile, err := os.OpenFile(
		filename,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		panic("could not open log file: " + filename)
	}

	return log.New(logfile, "[read_lsp] ", log.Ldate|log.Ltime|log.Lshortfile)
}
