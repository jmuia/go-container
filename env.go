package main

import (
	"fmt"
	"os"
	"syscall"
)

func setEnv(c container) {
	setHostname(c.id())
	setPs1()
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
