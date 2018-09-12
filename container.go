package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
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
	fmt.Printf("Hello, I am container with pid %d\n", os.Getpid())

	// Do not participate in shared subtrees by recursively setting mounts under / to private.
	if err := syscall.Mount("none", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		panic(fmt.Sprintf("Error recursively settings mounts to private: %s\n", err))
	}

	cmd := exec.Command("/bin/bash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("Error running /bin/bash command: %s\n", err))
	}
}

func main() {
	fmt.Printf("Hello, I am main with pid %d\n", os.Getpid())

	cmd := reexec.Command("container")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS,
	}

	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("Error running reexec container command: %s\n", err))
	}
}
