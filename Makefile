BINARY := ir

.PHONY: build clean

build:
	go build -ldflags "-w -s" -v -o $(BINARY) ./cmd/ir

clean:
	rm -f $(BINARY)
