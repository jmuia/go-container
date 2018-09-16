package main

import (
	"flag"
	"fmt"
	"os"
)

func cliUsage() {
	fmt.Println("Usage: ./go-container [OPTIONS] <image name> <command>")
	flag.PrintDefaults()
}

func parseCliArgs() (container, networkConfig) {
	c := container{}
	net := networkConfig{}

	flag.Usage = cliUsage

	// Container.
	flag.StringVar(&c.containersDir, "c", "containers", "directory to store containers")
	flag.StringVar(&c.imagesDir, "i", "images", "directory to find container images")
	flag.IntVar(&c.cpuShares, "cpu", 0, "cpu shares (relative weight)")
	flag.StringVar(&c.memLimit, "mem", "", "memory limit in bytes; suffixes can be used")

	// NetworkConfig.
	flag.StringVar(&net.bridgeAddr, "bridge-addr", "10.10.10.1/24", "CIDR bridge address; replaces current if present")
	flag.StringVar(&net.containerVethAddr, "container-addr", "10.10.10.2/24", "CIDR container veth address")

	flag.Parse()

	if flag.NArg() < 2 {
		flag.Usage()
		os.Exit(1)
	}

	c.imageName = flag.Arg(0)
	c.command = flag.Args()[1:]

	return c, net
}
