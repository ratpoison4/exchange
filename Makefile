BIN=bin/exchange

all: install

install:
	go install github.com/z0rr0/exchange

debug: install
	cp config.example.json config.json
	$(GOPATH)/$(BIN)
