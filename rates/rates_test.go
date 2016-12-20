package rates

import (
	"log"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

const (
	configFile  = "config.example.json"
	packageName = "github.com/z0rr0/exchange"
)

var (
	logger = log.New(os.Stdout, "TEST: ", log.Ldate|log.Ltime|log.Lshortfile)
)

func getConfig() string {
	dirs := []string{os.Getenv("GOPATH"), "src"}
	dirs = append(dirs, strings.Split(packageName, "/")...)
	dirs = append(dirs, configFile)
	return path.Join(dirs...)
}

func TestNew(t *testing.T) {
	if _, err := New("/bad_file_path.json", logger); err == nil {
		t.Error("unexpected behavior")
	}
	cfgFile := getConfig()
	cfg, err := New(cfgFile, logger)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr() == "" {
		t.Error("empty address")
	}
}

func TestCfg_HandleTimeout(t *testing.T) {
	cfg, err := New(getConfig(), logger)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Timeout = 1
	if err := cfg.isValid(); err == nil {
		t.Error("uexpected behavior")
	}
	cfg.Timeout = 10
	if et := cfg.externalTimeout(); et != ((10 - 1) * time.Second) {
		t.Errorf("unxpected timeout: %v", et)
	}
	if et := cfg.HandleTimeout(); et != (10 * time.Second) {
		t.Errorf("unxpected timeout: %v", et)
	}
}

func TestCfg_GetCodes(t *testing.T) {
	cfg, err := New(getConfig(), logger)
	if err != nil {
		t.Fatal(err)
	}
	codes, err := cfg.GetCodes()
	if err != nil {
		t.Error(err)
	}
	if len(codes) == 0 {
		t.Error("unexpected behavior")
	}
}

func TestCfg_SetRequiredCodes(t *testing.T) {
	cfg, err := New(getConfig(), logger)
	if err != nil {
		t.Fatal(err)
	}
	requiredCodes := map[string][]string{
		"usd": {"$", "dollar"},
		"eur": {"€", "euro"},
	}
	err = cfg.SetRequiredCodes(requiredCodes)
	if err != nil {
		t.Error("unexpected behavior")
	}
}

func TestCfg_GetRates(t *testing.T) {
	cfg, err := New(getConfig(), logger)
	if err != nil {
		t.Fatal(err)
	}
	d, q := time.Now().UTC(), ""
	if _, err := cfg.GetRates(d, q); err == nil {
		t.Error("unexpected behavior")
	}
	requiredCodes := map[string][]string{
		"usd": {"$", "dollar"},
		"eur": {"€", "euro"},
	}
	err = cfg.SetRequiredCodes(requiredCodes)
	if err != nil {
		t.Error("unexpected behavior")
	}
	messages := []string{"100 dollars", "$1", "1 usd", "usd 1.5", "10 euros", "euro 10", "15.5 euros", "10 €"}
	for i, msg := range messages {
		info, err := cfg.GetRates(d, msg)
		if err != nil {
			t.Error(err)
		}
		if info == nil {
			t.Errorf("unexpected behavior [%v]", i)
		}
		logger.Println(info.Rates)
	}
	requiredCodes = map[string][]string{
		"bad": {"bad_value"},
	}
	err = cfg.SetRequiredCodes(requiredCodes)
	if err != nil {
		t.Error("unexpected behavior")
	}
	if _, err := cfg.GetRates(d, "1 bad_value"); err == nil {
		t.Error("unexpected behavior")
	}
}
