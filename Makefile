VERSION := $(shell ./tools/version.sh)

.PHONY: all release

all:
	go build

release: arm64 mac64 linux64 win64 appleSilicon

arm64:
	mkdir -p release
	env GOOS=linux GOARCH=arm64 go build
	tar cvfz release/har-$(VERSION)-linux-arm64.tar.gz har
	rm -f har

linux64:
	mkdir -p release
	env GOOS=linux GOARCH=amd64 go build
	tar cvfz release/har-$(VERSION)-linux-amd64.tar.gz har
	rm -f har

mac64:
	mkdir -p release
	env GOOS=darwin GOARCH=amd64 go build
	tar cvfz release/har-$(VERSION)-apple-amd64.tar.gz har
	rm -f har

appleSilicon:
	mkdir -p release
	env GOOS=darwin GOARCH=arm64 go build
	tar cvfz release/har-$(VERSION)-apple-arm64.tar.gz har
	rm -f har

win64:
	mkdir -p release
	env GOOS=windows GOARCH=amd64 go build
	zip release/har-$(VERSION)-windows-amd64.zip har.exe
	rm -f har.exe

clean:
	rm -f ./har
	rm -rf ./release
	rm -rf ./av
	rm -rf ./src

install: all
	cp ./har ~/.local/bin/
