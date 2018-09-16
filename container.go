package main

import (
	"fmt"
	"os"
	"path/filepath"
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
	reexec.Register("setup", setup)
	if reexec.Init() {
		os.Exit(0) // Do not run main() if we ran another function.
	}
}

type container struct {
	containerDir  string
	containerId   string
	containerRoot string
	imageName     string
	imageDir      string
	cpuShares     int
	memLimit      string
}

func (c container) id() string {
	return c.containerId
}

func (c container) root(subdirs ...string) string {
	return filepath.Join(append([]string{c.containerRoot}, subdirs...)...)
}

func setup() {
	c := container{}
	c.containerDir = os.Args[1]
	c.imageDir = os.Args[2]
	c.imageName = os.Args[3]
	c.memLimit = os.Args[5]

	cpuShares, err := strconv.Atoi(os.Args[4])
	if err != nil {
		panic(fmt.Sprintf("Error parsing cpu.shares: %s\n", err))
	}
	c.cpuShares = cpuShares

	containerId, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Sprintf("Error generating container uuid: %s\n", err))
	}
	c.containerId = containerId.String()
	c.containerRoot = filepath.Join(c.containerDir, c.containerId, "rootfs")

	// Do not participate in shared subtrees by recursively setting mounts under / to private.
	if err := syscall.Mount("none", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		panic(fmt.Sprintf("Error recursively settings mounts to private: %s\n", err))
	}

	setEnv(c)
	createCgroups(c)
	createContainerFilesystem(c)

	if err := syscall.Exec("/bin/sh", []string{"sh"}, os.Environ()); err != nil {
		panic(fmt.Sprintf("Error exec'ing /bin/sh: %s\n", err))
	}
}

func run(config runConfig) {
	cmd := reexec.Command(
		"setup",
		config.containersDir,
		config.imagesDir,
		config.imageName,
		strconv.Itoa(config.cpuShares),
		config.memLimit)

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
