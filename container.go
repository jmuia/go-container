package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
	"github.com/mholt/archiver"
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
	fmt.Printf("Hello, I am container with pid %d\n", os.Getpid())

	// Do not participate in shared subtrees by recursively setting mounts under / to private.
	if err := syscall.Mount("none", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		panic(fmt.Sprintf("Error recursively settings mounts to private: %s\n", err))
	}

	containerId, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Sprintf("Error generating container uuid: %s\n", err))
	}

	createContainerFilesystem("images", "alpine.tar.gz", "containers", containerId.String())

	cmd := exec.Command("/bin/sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("Error running /bin/bash command: %s\n", err))
	}
}

func createContainerFilesystem(imageDir string, imageName string, containerDir string, containerId string) {
	imagePath := filepath.Join(imageDir, imageName)
	containerRoot := filepath.Join(containerDir, containerId, "rootfs")

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		panic(fmt.Sprintf("Unable to locate image %s\n", imageName))
	}

	imageArchiver := archiver.MatchingFormat(imagePath)
	if imageArchiver == nil {
		panic(fmt.Sprintf("Unknown archive format for image %s\n", imageName))
	}

	if err := imageArchiver.Open(imagePath, containerRoot); err != nil {
		panic(fmt.Sprintf("Error extracting image %s: %s\n", imageName, err))
	}

	// Change the container's root file system.
	pivotRoot(containerRoot)
}

func pivotRoot(containerRoot string) {
	// bind mount containerRoot to itself to circumvent pivot_root requirement.
	if err := syscall.Mount(containerRoot, containerRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		panic(fmt.Sprintf("Error changing root file system (bind mount self): %s\n", err))
	}

	oldRoot := filepath.Join(containerRoot, "old_root")
	if err := os.MkdirAll(oldRoot, 0700); err != nil {
		panic(fmt.Sprintf("Error changing root file system (mkdir old_root): %s\n", err))
	}
	if err := syscall.PivotRoot(containerRoot, oldRoot); err != nil {
		panic(fmt.Sprintf("Error changing root file system (pivot_root): %s\n", err))
	}
	if err := os.Chdir("/"); err != nil {
		panic(fmt.Sprintf("Error changing root file system (chdir): %s\n", err))
	}

	// MNT_DETACH performs a lazy unmount while immediately disconnecting the
	// file system recursively. Allows us to proceed at the cost of leaving
	// the mount in an ambiguous state. See mount(2).
	if err := syscall.Unmount("/old_root", syscall.MNT_DETACH); err != nil {
		panic(fmt.Sprintf("Error changing root file system (unmount old_root): %s\n", err))
	}
	if err := os.RemoveAll("/old_root"); err != nil {
		panic(fmt.Sprintf("Error changing root file system (rmdir old_root): %s\n", err))
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
