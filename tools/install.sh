#! /bin/sh

# Figure out OS and ARCH
OS="`uname`"
ARCH="`uname -m`"
VERSION=v1.2.1
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

echo "Installing Har ${VERSION}..."
echo "  * Using https://github.com/sio2boss/har/releases/download/${VERSION}/har-${VERSION}-${OSARCH}.${FORMAT}"
curl -o /tmp/har.${FORMAT} -fsSL https://github.com/sio2boss/har/releases/download/${VERSION}/har-${VERSION}-${OSARCH}.${FORMAT}

if [ -e /tmp/har.${FORMAT} ]; then
    rm -f /usr/local/bin/har && \
      echo "  * Removing existing executable" && \
      cd /usr/local/bin/ && \
      tar xfz /tmp/har.${FORMAT} && \
      echo "  * Extracting" && \
      rm /tmp/har.${FORMAT} && \
      echo "  * Success!"
    exit
fi

echo "  * Failed due to some reason. Please try manually downloading har and copy the binary to /usr/local/bin"
