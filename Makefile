all:
	go install github.com/sio2boss/har

release: arm arm64 mac64 linux64 win64

arm:
	env GOOS=linux GOARCH=arm go build -v github.com/sio2boss/har
	tar cvfz har-v0.1.0-arm.tar.gz har
	rm -f har

arm64:
	env GOOS=linux GOARCH=arm64 go build -v github.com/sio2boss/har
	tar cvfz har-v0.1.0-arm64.tar.gz har
	rm -f har

mac64:
	env GOOS=darwin GOARCH=amd64 go build -v github.com/sio2boss/har
	tar cvfz har-v0.1.0-mac64.tar.gz har
	rm -f har

linux64:
	env GOOS=linux GOARCH=amd64 go build -v github.com/sio2boss/har
	tar cvfz har-v0.1.0-linux64.tar.gz har
	rm -f har

win64:
	env GOOS=windows GOARCH=amd64 go build -v github.com/sio2boss/har
	zip har-v0.1.0-win64.zip har.exe
	rm -f har.exe

test:
	./bin/har http://mirrors.advancedhosters.com/apache/cassandra/2.1.14/apache-cassandra-2.1.14-bin.tar.gz

clean:
	rm -rf ./bin/har

install: all
	cp ./bin/har /usr/local/bin/