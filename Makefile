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
	# go tool cover -html=github.com/z0rr0/exchange/rates/coverage.out
	# go tool trace <package_path>/<package_name>.test github.com/z0rr0/exchange/rates/trace.out
	go test -race -v -cover -coverprofile=coverage.out -trace trace.out github.com/z0rr0/exchange/rates

docker: lint
	cp $(GOPATH)/$(BIN) ./
	docker build -t $(PROGRAM) .

clean:
	rm -f $(PROGRAM)
	rm -f $(GOPATH)/$(BIN)
