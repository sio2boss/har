#! /bin/sh

# Figure out OS and ARCH
OS="`uname`"
ARCH="`uname -m`"
VERSION=v0.1.0
OSARCH=
FORMAT=tar.gz
case $OS in
  'Linux')
    case $ARCH in
        'x86_64')
            OSARCH='linux64'
            ;;
        'armv8')
            OSARCH='arm64'
            ;;
        'armv7l')
            OSARCH='arm'
            ;;
        *)
            OSARCH='linux32'
    esac
    ;;
  'Darwin')
    OSARCH='mac64'
    ;;
  *) ;;
esac

echo "https://github.com/sio2boss/har/releases/download/v0.1.0/har-${VERSION}-${OSARCH}.${FORMAT}"
curl -o /tmp/har.${FORMAT} -fsSL https://github.com/sio2boss/har/releases/download/v0.1.0/har-${VERSION}-${OSARCH}.${FORMAT}

if [ -e /tmp/har.${FORMAT} ]; then
    rm -f /usr/local/bin/har && cd /usr/local/bin/ && tar xfz /tmp/har.${FORMAT} && rm /tmp/har.${FORMAT} && echo "done"
    exit
fi

echo "Failed due to some reason. Please try manually downloading har and copy the binary to /usr/local/bin"