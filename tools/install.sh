#! /bin/sh

# Figure out OS and ARCH
OS="`uname`"
ARCH="`uname -m`"
VERSION=v1.2.2
OSARCH=
FORMAT=tar.gz
case $OS in
  'Linux')
    case $ARCH in
        'x86_64')
            OSARCH='linux-amd64'
            ;;
        'armv8')
            OSARCH='linux-arm64'
            ;;
    esac
    ;;
  'Darwin')
    case $ARCH in
        'x86_64')
            OSARCH='apple-amd64'
            ;;
        'arm64')
            OSARCH='apple-arm64'
            ;;
    esac
    ;;
  'Windows')
    case $ARCH in
        'x86_64')
            OSARCH='windows-amd64'
            ;;
        'arm64')
            OSARCH='windows-arm64'
            ;;
    esac
    ;;
  *) ;;
esac


echo "Installing Har ${VERSION}..."
echo "  * Using https://github.com/sio2boss/har/releases/download/${VERSION}/har-${VERSION}-${OSARCH}.${FORMAT}"
curl -o /tmp/har.${FORMAT} -fsSL https://github.com/sio2boss/har/releases/download/${VERSION}/har-${VERSION}-${OSARCH}.${FORMAT}

if [ -e /tmp/har.${FORMAT} ]; then
    mkdir -p ~/.local/bin
    rm -f ~/.local/bin/har && \
      echo "  * Removing existing executable" && \
      tar xfz /tmp/har.${FORMAT} -C ~/.local/bin/ && \
      echo "  * Extracting" && \
      rm /tmp/har.${FORMAT} && \
      echo "  * Success!" && \
    exit
fi

echo "  * Failed due to some reason. Please try manually downloading har and copy the binary to ~/.local/bin"
