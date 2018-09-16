package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/mholt/archiver"
)

func pivotRoot(c container) {
	// Bind mount containerRoot to itself to circumvent pivot_root requirement.
	if err := syscall.Mount(c.root(), c.root(), "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		panic(fmt.Sprintf("Error changing root file system (bind mount self): %s\n", err))
	}

	if err := os.MkdirAll(c.root("old_root"), 0700); err != nil {
		panic(fmt.Sprintf("Error changing root file system (mkdir old_root): %s\n", err))
	}
	if err := syscall.PivotRoot(c.root(), c.root("old_root")); err != nil {
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

	// Container's root is now /.
	c.containerRoot = "/"
}

func createRootFs(c container) {
	exists, err := fileExists(c.image())
	if err != nil {
		panic(fmt.Sprintf("Unable to locate image %s\n", c.imageName))
	}

	if !exists {
		extractImage(c)
	}

	if err := os.MkdirAll(c.root(), 0755); err != nil {
		panic(fmt.Sprintf("Error creating overlay: %s\n", err))
	}
	if err := os.MkdirAll(c.container("workdir"), 0755); err != nil {
		panic(fmt.Sprintf("Error creating overlay: %s\n", err))
	}

	overlayData := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", c.image(), c.root(), c.container("workdir"))

	if err := syscall.Mount("overlay", c.root(), "overlay", syscall.MS_NODEV, overlayData); err != nil {
		panic(fmt.Sprintf("Error creating overlay: %s\n", err))
	}
}

func extractImage(c container) {
	imageArchive := findImageArchive(c)
	imageArchiver := archiver.MatchingFormat(imageArchive)
	if imageArchiver == nil {
		panic(fmt.Sprintf("Unknown archive format for image %s\n", c.imageName))
	}

	if err := imageArchiver.Open(imageArchive, c.image()); err != nil {
		panic(fmt.Sprintf("Error extracting image %s: %s\n", c.imageName, err))
	}
}

func findImageArchive(c container) string {
	matches, err := filepath.Glob(c.image() + ".*")
	if err != nil || len(matches) == 0 {
		panic(fmt.Sprintf("Unable to locate image %s\n", c.imageName))
	}
	if len(matches) != 1 {
		panic(fmt.Sprintf("Ambiguous image %s; multiple images match\n", c.imageName))
	}
	return matches[0]
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
