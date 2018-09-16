package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

func createCgroups(containerId string, cpuShares int, memLimit string) {
	cpuGroup := map[string]string{
		"cgroup.procs":      strconv.Itoa(os.Getpid()),
		"notify_on_release": "1",
	}
	if cpuShares > 0 {
		cpuGroup["cpu.shares"] = strconv.Itoa(cpuShares)
	}

	memGroup := map[string]string{
		"cgroup.procs":      strconv.Itoa(os.Getpid()),
		"notify_on_release": "1",
	}
	if memLimit != "" {
		memGroup["memory.limit_in_bytes"] = memLimit
	}

	createCgroup("cpu", containerId, cpuGroup)
	createCgroup("memory", containerId, memGroup)
}

func createCgroup(kind string, containerId string, fileData map[string]string) {
	cgroupDir := makeCgroupDir(kind, containerId)
	for file, data := range fileData {
		err := ioutil.WriteFile(filepath.Join(cgroupDir, file), []byte(data), 0700)
		if err != nil {
			panic(fmt.Sprintf("Error writing cgroup %s/%s: %s\n", kind, file, err))
		}
	}

}

func makeCgroupDir(kind string, containerId string) string {
	cgroupDir := filepath.Join("/", "sys", "fs", "cgroup", kind, "go_containers", containerId)
	if err := os.MkdirAll(cgroupDir, 0755); err != nil {
		panic(fmt.Sprintf("Error creating cgroup dir (mkdir %s): %s\n", cgroupDir, err))
	}
	return cgroupDir
}
