package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/mholt/archiver"
)

func createContainerFilesystem(imageDir string, imageName string, containerDir string, containerId string) {
	imagePath := findImage(imageDir, imageName)
	containerRoot := filepath.Join(containerDir, containerId, "rootfs")

	imageArchiver := archiver.MatchingFormat(imagePath)
	if imageArchiver == nil {
		panic(fmt.Sprintf("Unknown archive format for image %s\n", imageName))
	}

	if err := imageArchiver.Open(imagePath, containerRoot); err != nil {
		panic(fmt.Sprintf("Error extracting image %s: %s\n", imageName, err))
	}

	fmt.Printf("Created container rootfs: %s\n", containerRoot)

	// Mount special file systems per Open Containers spec.
	// https://github.com/opencontainers/runtime-spec/blob/master/config-linux.md
	mountSpecialFilesystems(containerRoot)

	// Change the container's root file system.
	pivotRoot(containerRoot)
}

func findImage(imageDir string, imageName string) string {
	matches, err := filepath.Glob(filepath.Join(imageDir, imageName) + ".*")
	if err != nil || len(matches) == 0 {
		panic(fmt.Sprintf("Unable to locate image %s\n", imageName))
	}
	if len(matches) != 1 {
		panic(fmt.Sprintf("Ambiguous image %s; multiple images match\n", imageName))
	}

	imagePath := matches[0]

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		panic(fmt.Sprintf("Unable to locate image %s\n", imageName))
	}

	return imagePath
}

func mountSpecialFilesystems(containerRoot string) {
	mustMount("proc", filepath.Join(containerRoot, "proc"), "proc", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC, "")
	mustMount("sysfs", filepath.Join(containerRoot, "sys"), "sysfs", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC, "")

	// With devtmpfs, kernel will automatically create device nodes.
	// It cannot be used in a user namespace, but our containers
	// don't use that feature currently. If using a separate user
	// namespace, we'll mount tmpfs on /dev and create devices manually.
	mustMount("devtmpfs", filepath.Join(containerRoot, "dev"), "devtmpfs", syscall.MS_NOSUID, "mode=0755")

	mustMount("devpts", filepath.Join(containerRoot, "dev", "pts"), "devpts", syscall.MS_NOSUID|syscall.MS_NOEXEC, "newinstance")
	mustMount("tmpfs", filepath.Join(containerRoot, "dev", "shm"), "tmpfs", syscall.MS_NOSUID|syscall.MS_NODEV, "")

}

func mustMount(source string, target string, fstype string, flags uintptr, data string) {
	if err := os.MkdirAll(target, os.ModeDir); err != nil {
		panic(fmt.Sprintf("Error creating mount dir (mkdir %s) in container: %s\n", target, err))
	}

	if err := syscall.Mount(source, target, fstype, flags, data); err != nil {
		panic(fmt.Sprintf("Error mounting %s in container: %s\n", target, err))
	}
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
