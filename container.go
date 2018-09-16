package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
	"github.com/satori/go.uuid"
)

func init() {
	// Go doesn't expose the fork system call. This appears to be due
	// to the fact that the Go runtime is multi-threaded and forking a
	// multi-threaded process is difficult, error prone, and unreliable.
	// https://stackoverflow.com/questions/28370646/how-do-i-fork-a-go-process
	// https://forum.golangbridge.org/t/function-fork-analog-to-go/6782/7

	// Instead, we re-exec the same process (using /proc/self/exe) with
	// a different argv[0] to indicate which code path to follow.
	reexec.Register("container", container)
	if reexec.Init() {
		os.Exit(0) // Do not run main() if we ran another function.
	}
}

func container() {
	containerDir, imageDir, imageName := os.Args[1], os.Args[2], os.Args[3]
	cpuShares, err := strconv.Atoi(os.Args[4])
	if err != nil {
		panic(fmt.Sprintf("Error parsing cpu.shares: %s\n", err))
	}

	// Do not participate in shared subtrees by recursively setting mounts under / to private.
	if err := syscall.Mount("none", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		panic(fmt.Sprintf("Error recursively settings mounts to private: %s\n", err))
	}

	containerId, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Sprintf("Error generating container uuid: %s\n", err))
	}

	setEnv(containerId.String())
	createCgroups(containerId.String(), cpuShares)
	createContainerFilesystem(imageDir, imageName, containerDir, containerId.String())

	if err := syscall.Exec("/bin/sh", []string{"sh"}, os.Environ()); err != nil {
		panic(fmt.Sprintf("Error exec'ing /bin/sh: %s\n", err))
	}
}

func run(config runConfig) {
	cmd := reexec.Command("container", config.containersDir, config.imagesDir, config.imageName, strconv.Itoa(config.cpuShares))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// TODO: cmd.Env = []string{}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNET,
	}

	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("Error running reexec container command: %s\n", err))
	}
	fmt.Printf("%d exited ok\n", cmd.Process.Pid)
}
