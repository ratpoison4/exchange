// Package main is client program for github.com/z0rr0/exchange service.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/z0rr0/exchange/rates"
)

const (
	name           = "ExchangeClient"
	serviceURL     = "https://r.lus.su"
	serviceTimeout = 3000
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

	loggerInfo = log.New(os.Stdout, fmt.Sprintf("INFO [%v]: ", name), log.Ldate|log.Lmicroseconds|log.Lshortfile)
)

func request(serviceHost, query, date, userAgent string, timeout time.Duration, debug bool) (*rates.Info, error) {
	var resp *http.Response
	if debug {
		start := time.Now()
		loggerInfo.Println("start")
		defer func() {
			loggerInfo.Printf("end, duration %v\n", time.Since(start))
		}()
	}
	params := url.Values{}
	params.Add("q", query)
	params.Add("d", date)

	req, err := http.NewRequest("GET", fmt.Sprintf("%v/?%v", serviceHost, params.Encode()), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", userAgent)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req = req.WithContext(ctx)

	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	client := &http.Client{Transport: tr}

	ec := make(chan error)
	defer close(ec)

	go func() {
		resp, err = client.Do(req)
		ec <- err
	}()
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("timed out (%v)", timeout)
	case err := <-ec:
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()
	if status := resp.StatusCode; status != http.StatusOK {
		return nil, fmt.Errorf("not ok status response: %v", status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	info := &rates.Info{}
	err = json.Unmarshal(body, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func main() {
	debug := flag.Bool("debug", false, "debug mode")
	version := flag.Bool("version", false, "show version")
	timeoutUint := flag.Uint("timeout", serviceTimeout, "timeout (milliseconds)")
	query := flag.String("q", "1 rub", "query")
	service := flag.String("service", serviceURL, "service URL")
	date := flag.String("d", time.Now().UTC().Format("2006-01-02"), "default current UTC date")
	flag.Parse()

	if *version {
		fmt.Printf("\tVersion: %v\n\tRevision: %v\n\tBuild date: %v\n\tGo version: %v\n",
			Version, Revision, Date, GoVersion)
		flag.PrintDefaults()
		return
	}
	timeout := time.Duration(*timeoutUint) * time.Millisecond

	info, err := request(*service, *query, *date, fmt.Sprintf("%v/%v", name, Version), timeout, *debug)
	if err != nil {
		if *debug {
			loggerInfo.Fatal(err)
		} else {
			fmt.Printf("ERROR: %v\n", err)
		}
		return
	}
	fmt.Println(info)
}
