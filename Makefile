BIN=bin/exchange
VERSION=`bash version.sh`


all: install

install:
	go install -ldflags "$(VERSION)" github.com/z0rr0/exchange

run: install
	cp config.example.json config.json
	$(GOPATH)/$(BIN)
