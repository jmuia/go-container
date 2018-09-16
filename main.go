package main

import (
	"flag"
	"fmt"
	"os"
)

type runConfig struct {
	containersDir string
	imagesDir     string
	imageName     string
	cpuShares     int
}

func cliUsage() {
	fmt.Printf("Usage: %s [OPTIONS] <image name>\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var config runConfig

	flag.Usage = cliUsage

	flag.StringVar(&config.containersDir, "c", "containers", "directory to store containers")
	flag.StringVar(&config.imagesDir, "i", "images", "directory to find container images")

	flag.IntVar(&config.cpuShares, "cpu-shares", 0, "cpu shares (relative weight)")

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	config.imageName = flag.Arg(0)

	run(config)
}
