.PHONY: build run clean

build:
	go build -o uber-exporter ./cmd/uber-exporter

run: build
	./uber-exporter

clean:
	rm -f uber-exporter

-include Makefile.ctx
