package main

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// Create special devices per Open Containers spec.
// https://github.com/opencontainers/runtime-spec/blob/master/config-linux.md

// Device documentation: https://www.kernel.org/doc/Documentation/admin-guide/devices.txt
func makeDevices(c container) {
	// Standard file descriptors.
	err := os.Symlink(c.root("proc", "self", "fd"), c.root("dev", "fd"))
	if err != nil {
		panic(fmt.Sprintf("Error symlink fd: %s\n", err))
	}

	for i, dev := range []string{"stdin", "stdout", "stderr"} {
		err := os.Symlink(fmt.Sprintf("/proc/self/fd/%d", i), c.root("dev", dev))
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
		device := int(unix.Mkdev(dev.major, dev.minor))

		err := syscall.Mknod(c.root("dev", dev.name), dev.kind|dev.perms, device)
		if err != nil {
			panic(fmt.Sprintf("Error mknod %s: %s\n", dev.name, err))
		}

	}

	// Bind mount console.
	_, err = os.Create(c.root("dev", "console"))
	if err != nil {
		panic(fmt.Sprintf("Error bind mount console: %s\n", err))
	}

	err = syscall.Mount(c.root("dev", "pts", "1"), c.root("dev", "console"), "", syscall.MS_BIND, "")
	if err != nil {
		panic(fmt.Sprintf("Error bind mount console: %s\n", err))
	}
}
