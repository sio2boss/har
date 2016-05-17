all:
	go install github.com/sio2boss/har

test:
	./bin/har http://mirrors.advancedhosters.com/apache/cassandra/2.1.14/apache-cassandra-2.1.14-bin.tar.gz

clean:
	rm -rf ./bin

install: all
	cp ./bin/har /usr/local/bin/