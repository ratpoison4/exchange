PROGRAM=exchange
BIN=bin/exchange
VERSION=`bash version.sh`
SOURCEDIR=src/github.com/z0rr0/exchange
CONTAINER=container.sh


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

test: lint
	# go tool cover -html=ratest_coverage.out
	# go tool trace ratest.test ratest_trace.out
	go test -race -v -cover -coverprofile=ratest_coverage.out -trace ratest_trace.out github.com/z0rr0/exchange/rates

clients:
	go vet github.com/z0rr0/exchange/client
	golint github.com/z0rr0/exchange/client
	go install -ldflags "$(VERSION)" github.com/z0rr0/exchange/client

docker: lint
	bash $(CONTAINER)
	docker build -t $(PROGRAM) .

clean:
	rm -f $(PROGRAM) $(GOPATH)/$(BIN)
	rm -rf $(GOPATH)/$(SOURCEDIR)/*.out
