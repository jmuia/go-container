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

func parseCliArgs() container {
	c := container{}

	flag.Usage = cliUsage

	flag.StringVar(&c.containersDir, "c", "containers", "directory to store containers")
	flag.StringVar(&c.imagesDir, "i", "images", "directory to find container images")
	flag.IntVar(&c.cpuShares, "cpu", 0, "cpu shares (relative weight)")

	flag.StringVar(&c.memLimit, "mem", "", "memory limit in bytes; suffixes can be used")

	flag.Parse()

	if flag.NArg() < 2 {
		flag.Usage()
		os.Exit(1)
	}

	c.imageName = flag.Arg(0)
	c.command = flag.Args()[1:]

	return c
}
