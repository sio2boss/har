VERSION := $(shell ./tools/version.sh)

.PHONY: all release

all:
	go build

release: linux_arm64 linux_amd64 apple_amd64 apple_arm64 win_amd64 win_arm64

linux_arm64:
	mkdir -p release
	env GOOS=linux GOARCH=arm64 go build
	tar cvfz release/har-$(VERSION)-linux-arm64.tar.gz har
	rm -f har

linux_amd64:
	mkdir -p release
	env GOOS=linux GOARCH=amd64 go build
	tar cvfz release/har-$(VERSION)-linux-amd64.tar.gz har
	rm -f har

apple_amd64:
	mkdir -p release
	env GOOS=darwin GOARCH=amd64 go build
	tar cvfz release/har-$(VERSION)-apple-amd64.tar.gz har
	rm -f har

apple_arm64:
	mkdir -p release
	env GOOS=darwin GOARCH=arm64 go build
	tar cvfz release/har-$(VERSION)-apple-arm64.tar.gz har
	rm -f har

win_amd64:
	mkdir -p release
	env GOOS=windows GOARCH=amd64 go build
	zip release/har-$(VERSION)-windows-amd64.zip har.exe
	rm -f har.exe

win_arm64:
	mkdir -p release
	env GOOS=windows GOARCH=arm64 go build
	zip release/har-$(VERSION)-windows-arm64.zip har.exe
	rm -f har.exe

clean:
	rm -f ./har
	rm -rf ./release
	rm -rf ./av
	rm -rf ./src

install: all
	cp ./har ~/.local/bin/
