# exchange

[![GoDoc](https://godoc.org/github.com/z0rr0/exchange?status.svg)](https://godoc.org/github.com/z0rr0/exchange)

It's a web service to return rate exchange for dollars/euros/rubles by incoming comma-separated text messages.

For example:

```
http -b -f POST localhost:8070 q="5 usd, 20 €, 100 рублей" d="2016-12-08"
```

```json
{
    "date": "2016-12-08", 
    "rates": [
        {
            "msg": "5 usd", 
            "rate": {
                "eur": 4.67, 
                "rub": 319.56, 
                "usd": 5
            }
        }, 
        {
            "msg": "20 €", 
            "rate": {
                "eur": 20, 
                "rub": 1370, 
                "usd": 21.44
            }
        }, 
        {
            "msg": "100 рублей", 
            "rate": {
                "eur": 1.46, 
                "rub": 100, 
                "usd": 1.56
            }
        }
    ]
}

```