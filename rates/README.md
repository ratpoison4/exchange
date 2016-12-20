# rates

[![GoDoc](https://godoc.org/github.com/z0rr0/exchange/rates?status.svg)](https://godoc.org/github.com/z0rr0/exchange/rates)

The package contains methods to get currencies exchange rates from Russian Central Bank [https://www.cbr.ru](https://www.cbr.ru).

```go
cfg, err := New(getConfig(), logger)
if err != nil {
        log.Fatal(err)
}
// currencies aliases
requiredCodes := map[string][]string{
        "usd": {"$", "dollar"},
        "eur": {"€", "euro"},
}
err = cfg.SetRequiredCodes(requiredCodes)
if err != nil {
        log.Fatal(err)
}
info, err := cfg.GetRates(d, "15.5 euro, 100$")
if err != nil {
        log.Fatal(err)
}
// info.Rates
// [{15.5 euro map[eur:15.5 usd:16.19]}, {100$ map[usd:100 eur:95.77]}]
```