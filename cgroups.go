package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

func createCgroups(containerId string, cpuShares int) {
	cpuCgroupDir := makeCgroupDir("cpu", containerId)

	// Add our pid to the cgroup.
	tasksPath := filepath.Join(cpuCgroupDir, "cgroup.procs")
	err := ioutil.WriteFile(tasksPath, []byte(strconv.Itoa(os.Getpid())), 0700)
	if err != nil {
		panic(fmt.Sprintf("Error adding self to cgroup.procs: %s\n", err))
	}

	// Set the cpu.shares.
	if cpuShares > 0 {
		cpuSharesPath := filepath.Join(cpuCgroupDir, "cpu.shares")
		err := ioutil.WriteFile(cpuSharesPath, []byte(strconv.Itoa(cpuShares)), 0700)
		if err != nil {
			panic(fmt.Sprintf("Error setting cpu.shares: %s\n", err))
		}
	}

	// Removes the new cgroup in place after the container exits.
	notifyPath := filepath.Join(cpuCgroupDir, "notify_on_release")
	err = ioutil.WriteFile(notifyPath, []byte("1"), 0700)
	if err != nil {
		panic(fmt.Sprintf("Error setting cpu notify_on_release: %s\n", err))
	}
}

func makeCgroupDir(kind string, containerId string) string {
	cgroupDir := filepath.Join("/", "sys", "fs", "cgroup", kind, "go_containers", containerId)
	if err := os.MkdirAll(cgroupDir, 0755); err != nil {
		panic(fmt.Sprintf("Error creating cgroup dir (mkdir %s): %s\n", cgroupDir, err))
	}
	return cgroupDir
}
