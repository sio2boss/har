har
===

From the Swedish verb 'to have'.  We as developers and end users often download some file and then do some operation like extract it or install it or run it.  Har's goal is to just do all that from a single tool.  If you look at say a Dockerfile or Ansible or a Bash shell script, most of them have to figure out what compression was used and/or keep track of the filename used.  These are core problems we will address.

## Quick Install Instructions

Install with homebrew
```bash
brew install sio2boss/tap/har
```

## Modes

| Mode | Description                                        |
|------|----------------------------------------------------|
| i    | Download and install script                        |
| b    | Download and install binary file to /usr/local/bin |
| g    | Just Download                                      |
| x    | Download and extract                               |
| c    | Create binary installer from directory             |


## Usage

```sh
Usage:
  har (i|install) [--ruby|--python|--python3] [-y] [-s] [--sha1=<sum>] URL
  har (b|binary)  [-y] [-s] [--sha1=<sum>] URL [-O FILE]
  har (g|get)     [-y] [-s] [--sha1=<sum>] URL [-O FILE]
  har (x|extract) [-s] [--sha1=<sum>] URL [-C DIR]
  har (c|create)  DIR [-O FILE]
  har -h | --help
  har --version
```

## Use-Cases

### Download Stuff

Just grab files from the web and extract them, remove the archive.  This simplifies the av-shell binary install too

example usage:

    har x --silent https://github.com/BVLC/caffe/archive/rc3.zip
    har x https://github.com/sio2boss/av-shell/releases/download/2.1.0/av-shell-2.1.0-linux64.tar.gz -C ~/
    
or if you dont want to automatically extract (basically curl/wget but to a file):

    har g http://ftp.gnu.org/gnu/wget/wget2-2.0.0.tar.gz

### Install Stuff

There are a ton of examples on the internet where you download a file with curl and then run the script afterwards…brew, kops, av-shell.  But also there are apps that you just download and copy to /usr/local/bin and chmod like mc, kubectl, etc...

for the download, chmod, and move to ~/.local/bin style:

    har b https://github.com/kubernetes/kops/releases/download/$(curl -s https://api.github.com/repos/kubernetes/kops/releases/latest | grep tag_name | cut -d '"' -f 4)/kops-linux-amd64
    har b --sha1=d604417c2efba1413e2441f16de3be84d3d9b1ae https://storage.googleapis.com/kubernetes-release/release/v1.15.0/bin/linux/amd64/kubectl
    har b https://dl.min.io/client/mc/release/darwin-amd64/mc

for the run a script style:

    har i https://raw.githubusercontent.com/sio2boss/har/refs/heads/master/packs/git-hist.har
    har i https://raw.githubusercontent.com/sio2boss/har/master/tools/install.sh
    har i -—ruby https://raw.githubusercontent.com/Homebrew/install/master/install
    har i ./install.sh


## Development

```
go install gotest.tools/gotestsum@latest
gotestsum --format-icons hivis --format testname --hide-summary=all --watch
```
har
===

From the Swedish verb 'to have'.  We as developers and end users often download some file and then do some operation like extract it or install it or run it.  Har's goal is to just do all that from a single tool.  If you look at say a Dockerfile or Ansible or a Bash shell script, most of them have to figure out what compression was used and/or keep track of the filename used.  These are core problems we will address.

## Quick Install Instructions

This is the best way other folks on github have figured out how to download a golang based binary and install it.

    sh -c "$(curl -fsSL https://raw.githubusercontent.com/sio2boss/har/master/tools/install.sh)"

or if you already have har installed and want the update

    har i -y https://raw.githubusercontent.com/sio2boss/har/master/tools/install.sh

if you want to be able to run commands as root, then install with sudo

   sudo sh -c "$(curl -fsSL https://raw.githubusercontent.com/sio2boss/har/master/tools/install.sh)"

## Modes

| Mode | Description                                        |
|------|----------------------------------------------------|
| i    | Download and install script                        |
| b    | Download and install binary file to /usr/local/bin |
| g    | Just Download                                      |
| x    | Download and extract                               |
| c    | Create binary installer                            |


## Usage

```sh
Usage:
  har (i|install) [--ruby|--python|--python3] [-y] [-s] [--sha1=<sum>] URL
  har (b|binary)  [-y] [-s] [--sha1=<sum>] URL [-O FILE]
  har (g|get)     [-y] [-s] [--sha1=<sum>] URL [-O FILE]
  har (x|extract) [-s] [--sha1=<sum>] URL [-C DIR]
  har (c|create)  DIR [-O FILE]
  har -h | --help
  har --version
```

## Use-Cases

### Download Stuff

Just grab files from the web and extract them, remove the archive.  This simplifies the av-shell binary install too

example usage:

    har x --silent https://github.com/BVLC/caffe/archive/rc3.zip
    har x https://github.com/sio2boss/av-shell/releases/download/2.1.0/av-shell-2.1.0-linux64.tar.gz -C ~/
    
or if you dont want to automatically extract (basically curl/wget but to a file):

    har g http://ftp.gnu.org/gnu/wget/wget2-2.0.0.tar.gz

### Install Stuff

There are a ton of examples on the internet where you download a file with curl and then run the script afterwards…brew, kops, av-shell.  But also there are apps that you just download and copy to /usr/local/bin and chmod like mc, kubectl, etc...

for the download, chmod, and move to ~/.local/bin style:

    har b https://github.com/kubernetes/kops/releases/download/$(curl -s https://api.github.com/repos/kubernetes/kops/releases/latest | grep tag_name | cut -d '"' -f 4)/kops-linux-amd64
    har b --sha1=d604417c2efba1413e2441f16de3be84d3d9b1ae https://storage.googleapis.com/kubernetes-release/release/v1.15.0/bin/linux/amd64/kubectl
    har b https://dl.min.io/client/mc/release/darwin-amd64/mc

for the run a script style:

    har i https://raw.githubusercontent.com/sio2boss/har/refs/heads/master/packs/git-hist.har
    har i https://raw.githubusercontent.com/sio2boss/har/master/tools/install.sh
    har i -—ruby https://raw.githubusercontent.com/Homebrew/install/master/install
    har i ./install.sh


## Development

```
go install gotest.tools/gotestsum@latest
gotestsum --format-icons hivis --format testname --hide-summary=all --watch
```