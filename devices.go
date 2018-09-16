package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

// Create special devices per Open Containers spec.
// https://github.com/opencontainers/runtime-spec/blob/master/config-linux.md

// Device documentation: https://www.kernel.org/doc/Documentation/admin-guide/devices.txt
func makeDevices(containerRoot string) {
	// Standard file descriptors.
	fdSource := filepath.Join(containerRoot, "proc", "self", "fd")
	fdTarget := filepath.Join(containerRoot, "dev", "fd")
	err := os.Symlink(fdSource, fdTarget)
	if err != nil {
		panic(fmt.Sprintf("Error symlink fd: %s\n", err))
	}

	for i, dev := range []string{"stdin", "stdout", "stderr"} {
		devicePath := filepath.Join(containerRoot, "dev", dev)
		err := os.Symlink(fmt.Sprintf("/proc/self/fd/%d", i), devicePath)
		if err != nil {
			panic(fmt.Sprintf("Error symlink %s: %s\n", dev, err))
		}
	}

	// Special devices.
	devices := []struct {
		name  string
		kind  uint32
		perms uint32
		major uint32
		minor uint32
	}{
		{name: "null", kind: syscall.S_IFCHR, perms: 0666, major: 1, minor: 3},
		{name: "zero", kind: syscall.S_IFCHR, perms: 0666, major: 1, minor: 5},
		{name: "full", kind: syscall.S_IFCHR, perms: 0666, major: 1, minor: 7},
		{name: "random", kind: syscall.S_IFCHR, perms: 0666, major: 1, minor: 8},
		{name: "urandom", kind: syscall.S_IFCHR, perms: 0666, major: 1, minor: 9},
		{name: "tty", kind: syscall.S_IFCHR, perms: 0666, major: 5, minor: 0},
		{name: "ptmx", kind: syscall.S_IFCHR, perms: 0666, major: 5, minor: 2},
	}

	for _, dev := range devices {
		devicePath := filepath.Join(containerRoot, "dev", dev.name)
		device := int(unix.Mkdev(dev.major, dev.minor))

		err := syscall.Mknod(devicePath, dev.kind|dev.perms, device)
		if err != nil {
			panic(fmt.Sprintf("Error mknod %s: %s\n", dev.name, err))
		}

	}

	// Bind mount console.
	consoleSource := filepath.Join(containerRoot, "dev", "pts", "1")
	consoleTarget := filepath.Join(containerRoot, "dev", "console")
	_, err = os.Create(consoleTarget)
	if err != nil {
		panic(fmt.Sprintf("Error bind mount console: %s\n", err))
	}
	err = syscall.Mount(consoleSource, consoleTarget, "", syscall.MS_BIND, "")
	if err != nil {
		panic(fmt.Sprintf("Error bind mount console: %s\n", err))
	}
}
