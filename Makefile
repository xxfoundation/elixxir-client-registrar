.PHONY: update master release update_master update_release build clean version

version:
	go run main.go generate
	mv version_vars.go cmd/version_vars.go

clean:
	rm -rf vendor/
	go mod vendor

update:
	-GOFLAGS="" go get all

build:
	go build ./...
	go mod tidy

update_release:
	GOFLAGS="" go get gitlab.com/xx_network/primitives@release
	GOFLAGS="" go get gitlab.com/xx_network/crypto@release
	GOFLAGS="" go get gitlab.com/elixxir/crypto@release
	GOFLAGS="" go get gitlab.com/elixxir/comms@release

update_master:
	GOFLAGS="" go get gitlab.com/xx_network/primitives@master
	GOFLAGS="" go get gitlab.com/xx_network/crypto@master
	GOFLAGS="" go get gitlab.com/elixxir/crypto@release
	GOFLAGS="" go get gitlab.com/elixxir/comms@master

master: clean update_master build version

release: clean update_release build version

