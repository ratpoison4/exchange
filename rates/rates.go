// Package rates contains API methods handlers.
package rates

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru"
	"golang.org/x/net/html/charset"
)

const (
	currenciesCodesURL = "https://www.cbr.ru/scripts/XML_val.asp?d=0"
	currenciesRatesURL = "https://www.cbr.ru/scripts/XML_daily.asp"
)

// ResponseCodes is XML codes response.
type ResponseCodes struct {
	XMLName xml.Name   `xml:"Valuta"`
	Items   []CodeItem `xml:"Item"`
}

// CodeItem is currency code XML item.
type CodeItem struct {
	ID         string `xml:"ID,attr"`
	Name       string `xml:"Name"`
	EngName    string `xml:"EngName"`
	Nominal    uint   `xml:"Nominal"`
	ParentCode string `xml:"ParentCode"`
}

// ResponseRates is XML rates response.
type ResponseRates struct {
	XMLName xml.Name       `xml:"ValCurs"`
	Items   []CurrencyItem `xml:"Valute"`
}

// CurrencyItem is currency rate info.
type CurrencyItem struct {
	ID       string `xml:"ID,attr"`
	NumCode  string `xml:"NumCode"`
	CharCode string `xml:"CharCode"`
	Nominal  uint   `xml:"Nominal"`
	Name     string `xml:"Name"`
	Value    string `xml:"Value"`
}

// Info is rates' JSON struct response
type Info struct {
	Date  string     `json:"date"`
	Rates []RateItem `json:"rates"`
}

// RateItem is exchange rate item.
type RateItem struct {
	Msg  string             `json:"msg"`
	Rate map[string]float64 `json:"rate"`
}

// Cfg is rates' configuration settings.
type Cfg struct {
	Host      string `json:"host"`
	Port      uint   `json:"port"`
	CacheSize int    `json:"cache"`
	Timeout   int64  `json:"timeout"`
	Debug     bool   `json:"debug"`
	codes     map[string][]*regexp.Regexp
	cache     *lru.Cache
	logger    *log.Logger
}

// externalTimeout is external service timeout.
func (c *Cfg) externalTimeout() time.Duration {
	return time.Duration(c.Timeout-1) * time.Second
}

// isValid checks the settings are valid.
func (c *Cfg) isValid() error {
	// required 2 due to external timeout
	if c.Timeout < 2 {
		return errors.New("invalid timeout value")
	}
	return nil
}

// client returns HTTP client.
func (c *Cfg) client() *http.Client {
	tr := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		TLSHandshakeTimeout: c.externalTimeout(),
	}
	return &http.Client{Transport: tr}
}

// Addr return service's net address.
func (c *Cfg) Addr() string {
	return net.JoinHostPort(c.Host, fmt.Sprint(c.Port))
}

// HandleTimeout is service timeout
func (c *Cfg) HandleTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

// SetRequiredCodes sets required currencies char codes and their aliases.
// For example, {"USD": ["$", "dollar"], "RUB": ["руб", "rubles"]}
func (c *Cfg) SetRequiredCodes(codeNames map[string][]string) error {
	codes := make(map[string][]*regexp.Regexp)
	for code, names := range codeNames {
		namesRegexp := make([]*regexp.Regexp, (len(names)+1)*2)
		rg, err := regexp.Compile(fmt.Sprintf("(\\d+(\\.\\d+)?)\\s*%s", strings.ToLower(code)))
		if err != nil {
			return err
		}
		namesRegexp[0] = rg
		rg, err = regexp.Compile(fmt.Sprintf("%s\\s*(\\d+(\\.\\d+)?)", strings.ToLower(code)))
		if err != nil {
			return err
		}
		namesRegexp[1] = rg
		for i, name := range names {
			j := (i + 1) * 2
			namePattern := strings.ToLower(name)
			rg, err = regexp.Compile(fmt.Sprintf("(\\d+(\\.\\d+)?){1}\\s*%s", namePattern))
			if err != nil {
				return err
			}
			namesRegexp[j] = rg
			rg, err = regexp.Compile(fmt.Sprintf("%s\\s*(\\d+(\\.\\d+)?){1}", namePattern))
			if err != nil {
				return err
			}
			namesRegexp[j+1] = rg
		}
		codes[code] = namesRegexp
	}
	c.codes = codes
	return nil
}

// GetCodes returns available currencies codes.
func (c *Cfg) GetCodes() ([]CodeItem, error) {
	client := c.client()

	c.logger.Printf("start request to %v", currenciesCodesURL)
	defer func() {
		c.logger.Printf("done request to %v", currenciesCodesURL)
	}()

	resp, err := client.Get(currenciesCodesURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if statusCode := resp.StatusCode; statusCode != http.StatusOK {
		return nil, fmt.Errorf("not ok response: %v", statusCode)
	}
	codes := &ResponseCodes{}
	decoder := xml.NewDecoder(resp.Body)
	decoder.CharsetReader = charset.NewReaderLabel
	err = decoder.Decode(codes)
	if err != nil {
		return nil, err
	}
	return codes.Items, nil
}

// GetRates return currences rates info.
func (c *Cfg) GetRates(date time.Time, msg string) (*Info, int, error) {
	return &Info{}, http.StatusOK, nil
}

// New returns new rates configuration.
func New(filename string, logger *log.Logger) (*Cfg, error) {
	fullPath, err := filepath.Abs(strings.Trim(filename, " "))
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(fullPath)
	if err != nil {
		return nil, err
	}
	jsonData, err := ioutil.ReadFile(fullPath)
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
	if c.Debug {
		c.logger.SetOutput(os.Stdout)
	}
	c.cache = cache
	return c, err
}
