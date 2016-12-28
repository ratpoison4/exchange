// Package rates contains methods to get currencies exchange rates
// from Russian Central Bank https://www.cbr.ru
package rates

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

// RateError is error type during rates getting.
type RateError struct {
	HTTPCode int
	Msg      string
}

// Cfg is rates' configuration settings.
type Cfg struct {
	Host      string `json:"host"`
	Port      uint   `json:"port"`
	CacheSize int    `json:"cache"`
	Timeout   int64  `json:"timeout"`
	Debug     bool   `json:"debug"`
	timeout   time.Duration
	codes     map[string][]*regexp.Regexp
	userAgent string
	cache     *lru.Cache
	logger    *log.Logger
}

// parsedMsg is a stuct of parsed message.
type parsedMsg struct {
	msg      string
	currency string
	value    float64
}

// Error returns error message of RateError struct.
func (r *RateError) Error() string {
	return r.Msg
}

// externalTimeout is external service timeout.
func (c *Cfg) externalTimeout() time.Duration {
	// external timeout is less than service one (100ms)
	return time.Duration(c.Timeout*1000-100) * time.Millisecond
}

// isValid checks the settings are valid.
func (c *Cfg) isValid() error {
	// required 2 due to external timeout
	if c.Timeout < 1 {
		return errors.New("invalid timeout value")
	}
	return nil
}

// client returns HTTP client.
func (c *Cfg) client() *http.Client {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   10 * time.Second,
		MaxIdleConns:          100,
		IdleConnTimeout:       10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{Transport: tr}
}

// parseMsg returns corresponded
func (c *Cfg) parseMsg(messages []string) []parsedMsg {
	var nominal string
	result := make([]parsedMsg, len(messages))
	for j, m := range messages {
		message := strings.Trim(m, " ")
		result[j] = parsedMsg{msg: message}
		for currency, rgs := range c.codes {
			for i, rg := range rgs {
				if matches := rg.FindStringSubmatch(message); len(matches) == 4 {
					if i%2 == 0 {
						nominal = matches[1]
					} else {
						nominal = matches[2]
					}
					if value, err := strconv.ParseFloat(nominal, 64); err != nil {
						c.logger.Printf("parse float [%v] error: %v", nominal, err)
					} else {
						result[j].currency = currency
						result[j].value = value
						break
					}
				}
			}
			if result[j].value > 0 {
				// some currency already found
				break
			}
		}
	}
	return result
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
		rg, err := regexp.Compile(fmt.Sprintf("(\\d+(\\.\\d+)?)\\s*(%s)", strings.ToLower(code)))
		if err != nil {
			return err
		}
		namesRegexp[0] = rg
		rg, err = regexp.Compile(fmt.Sprintf("(%s)\\s*(\\d+(\\.\\d+)?)", strings.ToLower(code)))
		if err != nil {
			return err
		}
		namesRegexp[1] = rg
		for i, name := range names {
			j := (i + 1) * 2
			namePattern := strings.ToLower(name)
			rg, err = regexp.Compile(fmt.Sprintf("(\\d+(\\.\\d+)?){1}\\s*(%s)", namePattern))
			if err != nil {
				return err
			}
			namesRegexp[j] = rg
			rg, err = regexp.Compile(fmt.Sprintf("(%s)\\s*(\\d+(\\.\\d+)?){1}", namePattern))
			if err != nil {
				return err
			}
			namesRegexp[j+1] = rg
		}
		codes[strings.ToLower(code)] = namesRegexp
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

// dayRates gets currencies rates for requested day.
func (c *Cfg) dayRates(date time.Time) (*ResponseRates, error) {
	var resp *http.Response
	dateReq := date.Format("02/01/2006")
	if v, ok := c.cache.Get(dateReq); ok {
		return v.(*ResponseRates), nil
	}
	client := c.client()
	values := url.Values{}
	values.Add("date_req", dateReq)

	reqURL := fmt.Sprintf("%v?%v", currenciesRatesURL, values.Encode())
	c.logger.Printf("start request to %v", reqURL)
	defer func() {
		c.logger.Printf("done request to %v", reqURL)
	}()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", c.userAgent)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	req = req.WithContext(ctx)

	ec := make(chan error)
	defer close(ec)

	go func() {
		resp, err = client.Do(req)
		ec <- err
	}()
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("timed out (%v)", c.timeout)
	case err := <-ec:
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()
	if statusCode := resp.StatusCode; statusCode != http.StatusOK {
		return nil, fmt.Errorf("not ok response: %v", statusCode)
	}
	respRates := &ResponseRates{}
	decoder := xml.NewDecoder(resp.Body)
	decoder.CharsetReader = charset.NewReaderLabel
	err = decoder.Decode(respRates)
	if err != nil {
		return nil, err
	}
	c.cache.Add(dateReq, respRates)
	return respRates, nil
}

// reqRates prepares requested info.
func (c *Cfg) reqRates(date time.Time, messages []parsedMsg, info map[string]float64) ([]RateItem, error) {
	result := make([]RateItem, len(messages))
	for i, m := range messages {
		rate, ok := info[m.currency]
		if !ok {
			return nil, fmt.Errorf("unknown currency %v", m.currency)
		}
		// rub value
		value := rate * m.value
		result[i] = RateItem{Msg: m.msg, Rate: map[string]float64{}}
		// other values
		for currency := range c.codes {
			c.logger.Printf("value=%v, rate[%v]=%v", value, currency, info[currency])
			result[i].Rate[currency] = round(value/info[currency], 2)
		}
	}
	return result, nil
}

// GetRates return currences rates info.
func (c *Cfg) GetRates(date time.Time, msg string) (*Info, error) {
	if c.codes == nil {
		return nil, &RateError{HTTPCode: http.StatusInternalServerError, Msg: "uninitialized required codes"}
	}
	strDate := date.Format("2006-01-02")
	c.logger.Printf("start date=%v, msg=\"%v\"", strDate, msg)

	messages := strings.Split(strings.ToLower(msg), ",")
	if len(messages) == 0 {
		return &Info{Date: strDate, Rates: []RateItem{}}, nil
	}
	parsedMessages := c.parseMsg(messages)
	dayInfo, err := c.dayRates(date)
	if err != nil {
		return nil, &RateError{HTTPCode: http.StatusServiceUnavailable, Msg: "get daily rates"}
	}
	currencyInfo, err := currencyMap(dayInfo.Items)
	if err != nil {
		c.logger.Printf("currency map prepare: %v", err)
		return nil, &RateError{HTTPCode: http.StatusInternalServerError, Msg: "internal error"}
	}

	items, err := c.reqRates(date, parsedMessages, currencyInfo)
	if err != nil {
		c.logger.Printf("rates result prepare: %v", err)
		return nil, &RateError{HTTPCode: http.StatusBadRequest, Msg: "prepare rates error"}
	}
	return &Info{Date: strDate, Rates: items}, nil
}

// String returns string representation Info value.
func (i *Info) String() string {
	result := fmt.Sprintf("%v\n", i.Date)
	for _, rate := range i.Rates {
		result += fmt.Sprintf("\t%v\n", rate.Msg)
		for code, value := range rate.Rate {
			result += fmt.Sprintf("\t\t%v: %.3f\n", code, value)
		}
	}
	return result
}

// New returns new rates configuration.
func New(filename string, logger *log.Logger, userAgent string) (*Cfg, error) {
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
	c := &Cfg{logger: logger, userAgent: userAgent}
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
	c.timeout = time.Duration(c.Timeout) * time.Second
	return c, err
}

// currencyMap converts currencies response to float64 map.
func currencyMap(values []CurrencyItem) (map[string]float64, error) {
	result := make(map[string]float64)
	result["rub"] = 1.0
	for _, value := range values {
		floatStr := strings.Replace(value.Value, ",", ".", 1)
		v, err := strconv.ParseFloat(floatStr, 64)
		if err != nil {
			return nil, err
		}
		result[strings.ToLower(value.CharCode)] = v / float64(value.Nominal)
	}
	return result, nil
}

// round rounds positive val.
func round(val, places float64) float64 {
	const roundOn float64 = 0.5
	var round float64
	pow := math.Pow(10, places)
	digit := pow * val
	_, div := math.Modf(digit)

	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	return round / pow
}
