PROGRAM=exchange
BIN=bin/exchange
VERSION=`bash version.sh`


all: install

install:
	go install -ldflags "$(VERSION)" github.com/z0rr0/exchange

run: install
	cp config.example.json config.json
	$(GOPATH)/$(BIN)

lint: install
	go vet github.com/z0rr0/exchange/rates
	golint github.com/z0rr0/exchange/rates
	go vet github.com/z0rr0/exchange
	golint github.com/z0rr0/exchange

test: install
	go test -race -v -cover -coverprofile=rates_profile.out -trace rates_trace.out github.com/z0rr0/exchange/rates

docker: lint
	cp $(GOPATH)/$(BIN) ./
	docker build -t $(PROGRAM) .

clean:
	rm -f $(PROGRAM)
	rm -f $(GOPATH)/$(BIN)
