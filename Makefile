VERSION := $(shell ./tools/version.sh)

.PHONY: all test tests release homebrew update-version

update-version:
	@sed -i '' 's/"v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*"/"$(VERSION)"/' cmd/har/main.go

all: update-version
	go mod tidy
	go build -o har cmd/har/main.go 

test:
	go test ./...

tests: test

release: all linux_arm64 linux_amd64 apple_amd64 apple_arm64 win_amd64 win_arm64

linux_arm64:
	mkdir -p release
	env GOOS=linux GOARCH=arm64 go build -o har cmd/har/main.go 
	tar cvfz release/har-$(VERSION)-linux-arm64.tar.gz har
	rm -f har

linux_amd64:
	mkdir -p release
	env GOOS=linux GOARCH=amd64 go build -o har cmd/har/main.go 
	tar cvfz release/har-$(VERSION)-linux-amd64.tar.gz har
	rm -f har

apple_amd64:
	mkdir -p release
	env GOOS=darwin GOARCH=amd64 go build -o har cmd/har/main.go 
	tar cvfz release/har-$(VERSION)-apple-amd64.tar.gz har
	rm -f har

apple_arm64:
	mkdir -p release
	env GOOS=darwin GOARCH=arm64 go build -o har cmd/har/main.go
	tar cvfz release/har-$(VERSION)-apple-arm64.tar.gz har
	rm -f har

win_amd64:
	mkdir -p release
	env GOOS=windows GOARCH=amd64 go build -o har.exe cmd/har/main.go 
	zip release/har-$(VERSION)-windows-amd64.zip har.exe
	rm -f har.exe

win_arm64:
	mkdir -p release
	env GOOS=windows GOARCH=arm64 go build -o har.exe cmd/har/main.go
	zip release/har-$(VERSION)-windows-arm64.zip har.exe
	rm -f har.exe

homebrew:
	@echo "Updating Homebrew formula for version $(VERSION)..."
	./tools/update_homebrew_formula.sh $(VERSION)

clean:
	rm -f ./har
	rm -rf ./release

install: all
	cp ./har ~/.local/bin/
