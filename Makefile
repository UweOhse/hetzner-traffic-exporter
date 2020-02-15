PREFIX=/usr/local
VERSION=0.1

all: hetzner-traffic-exporter

hetzner-traffic-exporter: main.go version.go
	go build -o $@ $^

install: hetzner-traffic-exporter
	install -t $(PREFIX)/bin $^ 

version.go: Makefile version.in
	sed 's/VVVVV/$(VERSION)/g' <version.in >$@.t
	mv $@.t $@
