package rates

import (
	"log"
	"os"
	"path"
	"strings"
	"testing"
)

const (
	configFile  = "config.example.json"
	packageName = "github.com/z0rr0/exchange"
)

var (
	logger = log.New(os.Stdout, "TEST", log.Ldate|log.Ltime|log.Lshortfile)
)

func getConfig() string {
	dirs := []string{os.Getenv("GOPATH"), "src"}
	dirs = append(dirs, strings.Split(packageName, "/")...)
	dirs = append(dirs, configFile)
	return path.Join(dirs...)
}

func TestNew(t *testing.T) {
	cfgFile := getConfig()
	cfg, err := New(cfgFile, logger)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr() == "" {
		t.Error("empty address")
	}
}
