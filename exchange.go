// Package main runs web service to handle incoming requests and
// return Russian Central Bank rate exchange.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/z0rr0/exchange/rates"
)

const (
	// Name is a programm name
	Name = "Exchange"
	// Config is default configuration file name
	Config = "config.json"
	// interruptPrefix is constant prefix of interrupt signal
	interruptPrefix = "interrupt signal"
	// shutdownTimeout is connections' graceful shutdown timeout
	shutdownTimeout = time.Second * 2
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

	// requiredCodes are default required codes
	requiredCodes = map[string][]string{
		"USD": {"$", "dollar", "доллар"},
		"EUR": {"€", "euro", "евро"},
		"RUB": {"₽", "rub", "руб"},
	}
	// internal loggers
	loggerError = log.New(os.Stderr, fmt.Sprintf("ERROR [%v]: ", Name), log.Ldate|log.Ltime|log.Lshortfile)
	loggerInfo  = log.New(os.Stdout, fmt.Sprintf("INFO [%v]: ", Name), log.Ldate|log.Ltime|log.Lshortfile)
)

// help is help data structure
type help struct {
	D       string `json:"d"`
	Q       string `json:"q"`
	Comment string `json:"comment"`
}

// interrupt catches custom signals.
func interrupt(errc chan error) {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	errc <- fmt.Errorf("%v %v", interruptPrefix, <-c)
}

// helpFunc writes help info to ResponseWriter and returns HTTP status code.
func helpFunc(w http.ResponseWriter, r *http.Request, h *help) int {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(h); err != nil {
		code := http.StatusInternalServerError
		http.Error(w, http.StatusText(code), code)
		loggerError.Println(err.Error())
		return code
	}
	return http.StatusOK
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
		fmt.Printf("\tVersion: %v\n\tRevision: %v\n\tBuild date: %v\n\tGo version: %v\n",
			Version, Revision, Date, GoVersion)
		return
	}
	logger := log.New(ioutil.Discard, fmt.Sprintf("DEBUG [%v]: ", Name),
		log.Ldate|log.Lmicroseconds|log.Lshortfile)
	if *debug {
		logger.SetOutput(os.Stdout)
	}
	cfg, err := rates.New(*config, logger, fmt.Sprintf("%v/%v", Name, Version))
	if err != nil {
		loggerError.Fatalf("configuration error: %v", err)
	}
	err = cfg.SetRequiredCodes(requiredCodes)
	if err != nil {
		loggerError.Fatal(err)
	}
	h := &help{
		Q:       "query (default '1 rub')",
		D:       "date using format YYYY-MM-DD (default today) [optional]",
		Comment: "request get/post parameters",
	}
	server := &http.Server{
		Addr:           cfg.Addr(),
		Handler:        http.DefaultServeMux,
		ReadTimeout:    cfg.HandleTimeout(),
		WriteTimeout:   cfg.HandleTimeout(),
		MaxHeaderBytes: 1 << 20, // 1MB
		ErrorLog:       loggerError,
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var date time.Time
		start, code := time.Now(), http.StatusOK
		defer func() {
			loggerInfo.Printf("%-5v %v\t%-12v\t%v",
				r.Method,
				code,
				time.Since(start),
				r.URL.String(),
			)
		}()

		switch path := strings.TrimRight(r.URL.Path, "/"); {
		case path == "/help":
			code = helpFunc(w, r, h)
			return
		case path != "":
			code = http.StatusNotFound
			http.NotFound(w, r)
			return
		}

		query := r.FormValue("q")
		if query == "" {
			query = "1 rub"
		}
		if d := r.FormValue("d"); d != "" {
			date, err = time.Parse("2006-01-02", d)
			if err != nil {
				code = http.StatusBadRequest
				http.Error(w, "bad date format", code)
				return
			}
			if date.After(time.Now().UTC()) {
				code = http.StatusBadRequest
				http.Error(w, "bad date", code)
				return
			}
		} else {
			date = time.Now().UTC()
		}
		info, err := cfg.GetRates(date, query)
		if err != nil {
			rateError := err.(*rates.RateError)
			code = rateError.HTTPCode
			http.Error(w, err.Error(), code)
			loggerError.Println(err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		encoder := json.NewEncoder(w)
		err = encoder.Encode(info)
		if err != nil {
			code = http.StatusInternalServerError
			http.Error(w, http.StatusText(code), code)
			loggerError.Println(err.Error())
			return
		}
		// ok
	})
	errc := make(chan error)
	go interrupt(errc)
	go func() {
		errc <- server.ListenAndServe()
	}()
	loggerInfo.Printf("running: version=%v [%v %v debug=%v]\nListen: %v\n\n",
		Version, GoVersion, Revision, *debug || cfg.Debug, server.Addr)
	err = <-errc
	loggerInfo.Printf("termination: %v [%v] reason: %+v\n", Version, Revision, err)

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if msg := err.Error(); strings.HasPrefix(msg, interruptPrefix) {
		loggerInfo.Println("graceful shutdown")
		if err := server.Shutdown(ctx); err != nil {
			loggerError.Printf("graceful shutdown error: %v\n", err)
		}

	}
}
