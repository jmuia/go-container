package main

import (
	"fmt"
	"os"
	"path/filepath"
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
	containersDir string
	containerId   string
	containerRoot string
	imageName     string
	imagesDir     string
	cpuShares     int
	memLimit      string
	command       []string
}

func (c container) id() string {
	return c.containerId
}

func (c container) root(subdirs ...string) string {
	return filepath.Join(append([]string{c.containerRoot}, subdirs...)...)
}

func (c container) container(subdirs ...string) string {
	return filepath.Join(append([]string{c.containersDir, c.containerId}, subdirs...)...)
}

func (c container) image() string {
	return filepath.Join(c.imagesDir, c.imageName)
}

func setup() {
	c, _ := parseCliArgs()

	containerId := uuid.NewV4()
	c.containerId = containerId.String()
	c.containerRoot = filepath.Join(c.containersDir, c.containerId, "rootfs")

	// Do not participate in shared subtrees by recursively setting mounts under / to private.
	if err := syscall.Mount("none", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		panic(fmt.Sprintf("Error recursively settings mounts to private: %s\n", err))
	}

	createCgroups(c)
	createRootFs(c)
	mountSpecialFilesystems(c)
	makeDevices(c)
	pivotRoot(c)
	setupEnvironment(c)

	// TODO: wait for network to avoid race.

	if err := syscall.Exec(c.command[0], c.command, os.Environ()); err != nil {
		panic(fmt.Sprintf("Error exec'ing /bin/sh: %s\n", err))
	}
}

func main() {
	_, config := parseCliArgs()

	// Reexec changing only argv[0] to "setup".
	os.Args[0] = "setup"
	cmd := reexec.Command(os.Args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNET,
	}

	if err := cmd.Start(); err != nil {
		panic(fmt.Sprintf("Error running reexec container command: %s\n", err))
	}

	config.containerPid = cmd.Process.Pid
	setupNetwork(config)

	if err := cmd.Wait(); err != nil {
		panic(fmt.Sprintf("Error running reexec container command: %s\n", err))
	}
	fmt.Printf("%d exited ok\n", cmd.Process.Pid)
}
