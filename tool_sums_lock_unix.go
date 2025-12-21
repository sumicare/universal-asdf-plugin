//go:build !windows

package main

import "golang.org/x/sys/unix"

func lockToolSumsFile(fd int, exclusive bool) error {
	lockType := unix.LOCK_SH
	if exclusive {
		lockType = unix.LOCK_EX
	}

	return unix.Flock(fd, lockType)
}

func unlockToolSumsFile(fd int) error {
	return unix.Flock(fd, unix.LOCK_UN)
}
