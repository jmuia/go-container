# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/xenial64"

  config.vm.provision "shell", inline: <<-SHELL
    set -e

    # Running the container with `sudo` go will
    # look for the root's GOPATH.
    export GOPATH="/root/go"
    export PATH="$PATH:$GOPATH/bin"

    mkdir -p $GOPATH/bin
    mkdir -p $GOPATH/pkg
    mkdir -p $GOPATH/src

    add-apt-repository ppa:gophers/archive
    apt-get update -y
    apt-get install golang-1.10-go -y
    export PATH="/usr/lib/go-1.10/bin:$PATH"

    mkdir -p $GOPATH/src/github.com/jmuia/go-container/
    cp -r /vagrant/. $GOPATH/src/github.com/jmuia/go-container/
    cd $GOPATH/src/github.com/jmuia/go-container/

    # yikes!
    curl https://glide.sh/get | sh

    glide install -v
    CGO_ENABLED=0 GOOS=linux go build -a

  SHELL
end
