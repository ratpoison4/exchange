// Package rates contains API methods handlers.
package rates

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru"
)

// Cfg is rates' configuration settings.
type Cfg struct {
	Host      string `json:"host"`
	Port      uint   `json:"port"`
	CacheSize int    `json:"cache"`
	Timeout   int64  `json:"timeout"`
	cache     *lru.Cache
	logger    *log.Logger
}

// Addr return service's net address.
func (c *Cfg) Addr() string {
	return net.JoinHostPort(c.Host, fmt.Sprint(c.Port))
}

// HandleTimeout is service timeout
func (c *Cfg) HandleTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

// isValid checks the settings are valid.
func (c *Cfg) isValid() error {
	if c.Timeout <= 1 {
		return errors.New("invalid timeout value")
	}
	return nil
}

// New returns new rates configuration.
func New(filename string, logger *log.Logger) (*Cfg, error) {
	fullpath, err := filepath.Abs(strings.Trim(filename, " "))
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(fullpath)
	if err != nil {
		return nil, err
	}
	jsonData, err := ioutil.ReadFile(fullpath)
	if err != nil {
		return nil, err
	}
	c := &Cfg{logger: logger}
	err = json.Unmarshal(jsonData, c)
	if err != nil {
		return nil, err
	}
	err = c.isValid()
	if err != nil {
		return nil, err
	}
	cache, err := lru.New(c.CacheSize)
	if err != nil {
		return nil, err
	}
	c.cache = cache
	return c, err
}
