PREFIX=/usr/local
VERSION=0.1
G=`git rev-list HEAD | head -1`

all: hetzner-traffic-exporter

hetzner-traffic-exporter: main.go version.go
	go build -o $@ $^

install: hetzner-traffic-exporter
	install -t $(PREFIX)/bin $^ 

version.go: Makefile version.in
	sed -e 's/VVVVV/$(VERSION)/g' -e 's/GGGGG/'$G'/g' <version.in >$@.t
	mv $@.t $@

style:
	go vet main.go version.go
	errcheck
	staticcheck .
	gocritic check -enable='#diagnostic,#experimental,#performace,#style,#opionionated' ./...
	gosec ./...
