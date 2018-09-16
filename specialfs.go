package main

import (
	"fmt"
	"os"
	"syscall"
)

// Mount special file systems per Open Containers spec.
// https://github.com/opencontainers/runtime-spec/blob/master/config-linux.md
func mountSpecialFilesystems(c container) {
	mustMount("proc", c.root("proc"), "proc", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC, "")
	mustMount("sysfs", c.root("sys"), "sysfs", syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC, "")
	mustMount("tmpfs", c.root("dev"), "tmpfs", syscall.MS_NOSUID, "mode=0755")
	mustMount("devpts", c.root("dev", "pts"), "devpts", syscall.MS_NOSUID|syscall.MS_NOEXEC, "")
	mustMount("tmpfs", c.root("dev", "shm"), "tmpfs", syscall.MS_NOSUID|syscall.MS_NODEV, "")

}

func mustMount(source string, target string, fstype string, flags uintptr, data string) {
	if err := os.MkdirAll(target, 0755); err != nil {
		panic(fmt.Sprintf("Error creating mount dir (mkdir %s) in container: %s\n", target, err))
	}

	if err := syscall.Mount(source, target, fstype, flags, data); err != nil {
		panic(fmt.Sprintf("Error mounting %s in container: %s\n", target, err))
	}
}
