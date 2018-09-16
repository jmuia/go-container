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

	u, err := user.Current()
	if err != nil {
		panic(fmt.Sprintf("Unable to set env vars %s\n", err))
	}

	setHome(u)
	setHostname(c.id())
	mustSetEnv("USER", u.Username)
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

func setHome(u *user.User) {
	if os.Getuid() == 0 {
		mustSetEnv("HOME", "/root")
	} else {
		mustSetEnv("HOME", filepath.Join("/home", u.Username))
	}
}

func setHostname(containerId string) {
	if err := syscall.Sethostname([]byte(containerId)); err != nil {
		panic(fmt.Sprintf("Unable to set hostname %s\n", err))
	}
	mustSetEnv("HOSTNAME", containerId)
}
