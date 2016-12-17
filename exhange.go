// Package main runs web service
// to handle incoming rate exchange requests.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/z0rr0/exchange/rates"
)

const (
	// Name is a programm name
	Name = "Exchange"
	// Config is default configuration file name
	Config = "config.json"
)

var (
	// Version is a version from GIT tags
	Version = "0.0.0"
	// Revision - GIT revision number
	Revision = "git:000000"
	// Date - build date
	Date = "2016-01-01_01:01:01UTC"
	// GoVersion is runtime Go language version
	GoVersion = runtime.Version()

	loggerError = log.New(os.Stderr, fmt.Sprintf("ERROR [%v]: ", Name), log.Ldate|log.Ltime|log.Lshortfile)
	loggerInfo  = log.New(os.Stdout, fmt.Sprintf("INFO [%v]: ", Name), log.Ldate|log.Ltime|log.Lshortfile)
	loggerDebug = log.New(os.Stdout, fmt.Sprintf("DEBUG [%v]: ", Name), log.Ldate|log.Lmicroseconds|log.Lshortfile)
)

func interrupt() error {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	return fmt.Errorf("signal %v", <-c)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("abnormal termination [%v]: \n\t%v\n", Version, r)
		}
	}()
	debug := flag.Bool("debug", false, "debug mode")
	version := flag.Bool("version", false, "show version")
	config := flag.String("config", Config, "configuration file")
	flag.Parse()

	if *version {
		fmt.Printf("\tRevision: %v\n\tBuild date: %v\n\tGo version: %v\n", Revision, Date, GoVersion)
		return
	}
	logger := log.New(ioutil.Discard, "", 0)
	if *debug {
		logger = loggerDebug
	}
	cfg, err := rates.New(*config, logger)
	if err != nil {
		loggerError.Fatalf("configuration error: %v", err)
	}
	server := &http.Server{
		Addr:           cfg.Addr(),
		Handler:        http.DefaultServeMux,
		ReadTimeout:    cfg.HandleTimeout(),
		WriteTimeout:   cfg.HandleTimeout(),
		MaxHeaderBytes: 1 << 10, // 1kB
		ErrorLog:       loggerError,
	}
	errc := make(chan error)
	go func() {
		errc <- server.ListenAndServe()
	}()
	go func() {
		errc <- interrupt()
	}()
	loggerInfo.Printf("running: version=%v [%v %v debug=%v]\nListen: %v\n\n",
		Version, GoVersion, Revision, *debug, server.Addr)
	loggerInfo.Printf("termination: %v [%v] reason: %+v\n", Version, Revision, <-errc)
}
