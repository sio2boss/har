all: src/github.com/Sirupsen/logrus src/github.com/docopt/docopt-go src/gopkg.in/cheggaaa/pb.v1
	go build

src/github.com/Sirupsen/logrus:
	go get github.com/Sirupsen/logrus

src/github.com/docopt/docopt-go:
	go get github.com/docopt/docopt-go

src/gopkg.in/cheggaaa/pb.v1:
	go get gopkg.in/cheggaaa/pb.v1

release: arm arm64 mac64 linux64 win64

arm:
	mkdir -p release
	env GOOS=linux GOARCH=arm go build
	tar cvfz release/har-v0.1.1-arm.tar.gz har
	rm -f har

arm64:
	mkdir -p release
	env GOOS=linux GOARCH=arm64 go build
	tar cvfz release/har-v0.1.1-arm64.tar.gz har
	rm -f har

mac64:
	mkdir -p release
	env GOOS=darwin GOARCH=amd64 go build
	tar cvfz release/har-v0.1.1-mac64.tar.gz har
	rm -f har

linux64:
	mkdir -p release
	env GOOS=linux GOARCH=amd64 go build
	tar cvfz release/har-v0.1.1-linux64.tar.gz har
	rm -f har

win64:
	mkdir -p release
	env GOOS=windows GOARCH=amd64 go build
	zip release/har-v0.1.1-win64.zip har.exe
	rm -f har.exe

test:
	./har http://mirrors.advancedhosters.com/apache/cassandra/2.1.14/apache-cassandra-2.1.14-bin.tar.gz
	./har https://github.com/sio2boss/av.git

clean:
	rm -f ./har
	rm -rf ./release
	rm -rf ./av
	rm -rf ./src
	rm -rf apache-cassandra-2.1.14

install: all
	cp ./har /usr/local/bin/
