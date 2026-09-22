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
release:
	mkdir -p release
	gox -osarch="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 freebsd/amd64 openbsd/amd64 netbsd/amd64" -output="release/har-$(VERSION)-{{.OS}}-{{.Arch}}" ./cmd/har
	@for f in release/*; do \
		case "$$f" in *.tar.gz) continue ;; esac; \
		tar -czf "$$f.tar.gz" -C release "$$(basename "$$f")"; \
		rm -f "$$f"; \
	done 

homebrew:
	@echo "Updating Homebrew formula for version $(VERSION)..."
	./tools/update_homebrew_formula.sh $(VERSION)

clean:
	rm -f ./har
	rm -rf ./release

install: all
	cp ./har ~/.local/bin/
