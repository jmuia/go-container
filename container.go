package main

import (
	"flag"
	"fmt"
	"os"
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
	containerDir, imageDir, imageName := os.Args[1], os.Args[2], os.Args[3]

	// Do not participate in shared subtrees by recursively setting mounts under / to private.
	if err := syscall.Mount("none", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		panic(fmt.Sprintf("Error recursively settings mounts to private: %s\n", err))
	}

	containerId, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Sprintf("Error generating container uuid: %s\n", err))
	}

	setHostname(containerId.String())
	setPs1()
	createContainerFilesystem(imageDir, imageName, containerDir, containerId.String())

	if err := syscall.Exec("/bin/sh", []string{"sh"}, os.Environ()); err != nil {
		panic(fmt.Sprintf("Error exec'ing /bin/sh: %s\n", err))
	}
}

func setHostname(containerId string) {
	if err := syscall.Sethostname([]byte(containerId)); err != nil {
		panic(fmt.Sprintf("Unable to set hostname %s\n", err))
	}
}

func setPs1() {
	if err := os.Setenv("PS1", "$USER@$HOSTNAME$ "); err != nil {
		panic(fmt.Sprintf("Unable to set PS1%s\n", err))
	}
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

func cliUsage() {
	fmt.Printf("Usage: %s [OPTIONS] <image name>\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = cliUsage
	containersDirPtr := flag.String("c", "containers", "directory to store containers")
	imagesDirPtr := flag.String("i", "images", "directory to find container images")

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	imageName := flag.Arg(0)

	cmd := reexec.Command("container", *containersDirPtr, *imagesDirPtr, imageName)
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
