package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
)

func setupEnvironment(c container) {
	clearEnv()

	if os.Getuid() == 0 {
		mustSetEnv("USER", "root")
		mustSetEnv("HOME", "/root")
	} else {
		u, err := user.Current()
		if err != nil {
			panic(fmt.Sprintf("Unable to set env vars %s\n", err))
		}
		mustSetEnv("USER", u.Username)
		mustSetEnv("HOME", filepath.Join("/home", u.Username))
	}

	setHostname(c.id())
	mustSetEnv("PS1", "$USER@$HOSTNAME$ ")
	mustSetEnv("PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
}

func clearEnv() {
	for _, e := range os.Environ() {
		os.Unsetenv(strings.Split(e, "=")[0])
	}
}

func mustSetEnv(key string, value string) {
	if err := os.Setenv(key, value); err != nil {
		panic(fmt.Sprintf("Unable to set %s env var: %s\n", key, err))
	}
}

func setHostname(containerId string) {
	if err := syscall.Sethostname([]byte(containerId)); err != nil {
		panic(fmt.Sprintf("Unable to set hostname %s\n", err))
	}
	mustSetEnv("HOSTNAME", containerId)
}
