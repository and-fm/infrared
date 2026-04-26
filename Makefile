BINARY := ir

.PHONY: build clean

build:
	go build -ldflags "-w -s" -v -o $(BINARY) .

clean:
	rm -f $(BINARY)
