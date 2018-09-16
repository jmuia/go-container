package main

import (
	"fmt"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type networkConfig struct {
	hostVethAddr      string
	containerVethAddr string
	containerPid      int
}

type link struct {
	link netlink.Link
}

func (l link) up() {
	err := netlink.LinkSetUp(l.link)
	if err != nil {
		panic(fmt.Sprintf("Error bringing link %v up: %s\n", l.link, err))
	}
}

func (l link) addAddr(rawAddr string) {
	addr, err := netlink.ParseAddr(rawAddr)
	if err != nil {
		panic(fmt.Sprintf("Error parsing addr %s: %s\n", rawAddr, err))
	}
	err = netlink.AddrAdd(l.link, addr)
	if err != nil {
		panic(fmt.Sprintf("Error adding addr to %v: %s\n", l.link, err))
	}
}

func (l link) setNs(pid int) {
	err := netlink.LinkSetNsPid(l.link, pid)
	if err != nil {
		panic(fmt.Sprintf("Error moving link %v to pid ns %d: %s\n", l.link, pid, err))
	}
}

type netNsExecr struct{}

func (e netNsExecr) exec(pid int, work func()) {
	// Lock the OS Thread so we don't accidentally switch namespaces.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Save the current namespace.
	currentNs, err := netns.Get()
	defer currentNs.Close()
	if err != nil {
		panic(fmt.Sprintf("Error exec'ing net ns of pid %d: %s\n", pid, err))
	}

	// Get handle for pid's network namespace.
	pidNs, err := netns.GetFromPid(pid)
	if err != nil {
		panic(fmt.Sprintf("Error exec'ing net ns of pid %d: %s\n", pid, err))
	}

	// Switch namespace.
	err = netns.Set(pidNs)
	defer pidNs.Close()
	if err != nil {
		panic(fmt.Sprintf("Error exec'ing net ns of pid %d: %s\n", pid, err))
	}

	// Do the namespace-scoped work.
	work()

	// Switch back to the original namespace.
	err = netns.Set(currentNs)
	if err != nil {
		panic(fmt.Sprintf("Error exec'ing net ns of pid %d: %s\n", pid, err))
	}
}

func createVethPair(containerPid int) (hostVeth link, containerVeth link) {
	hostVethName := fmt.Sprintf("veth%dh", containerPid)
	containerVethName := fmt.Sprintf("veth%dc", containerPid)

	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = hostVethName

	hostVeth.link = &netlink.Veth{
		LinkAttrs: linkAttrs,
		PeerName:  containerVethName,
	}

	err := netlink.LinkAdd(hostVeth.link)
	if err != nil {
		panic(fmt.Sprintf("Error creating veth pair: %s\n", err))
	}

	containerVeth.link, err = netlink.LinkByName(containerVethName)
	if err != nil {
		panic(fmt.Sprintf("Error creating veth pair: %s\n", err))
	}

	return hostVeth, containerVeth
}

func setupNetwork(config networkConfig) {
	hostVeth, containerVeth := createVethPair(config.containerPid)

	hostVeth.up()
	hostVeth.addAddr(config.hostVethAddr)

	containerVeth.setNs(config.containerPid)

	execer := netNsExecr{}
	work := func() {
		// Ignore error case; bringing up lo is not that important.
		lo, err := netlink.LinkByName("lo")
		if err == nil {
			link{link: lo}.up()
		}

		containerVeth.addAddr(config.containerVethAddr)
		containerVeth.up()
	}
	execer.exec(config.containerPid, work)
}
