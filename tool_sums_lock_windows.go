//go:build windows

package main

import "golang.org/x/sys/windows"

func lockToolSumsFile(fd int, exclusive bool) error {
	flags := uint32(0)
	if exclusive {
		flags = windows.LOCKFILE_EXCLUSIVE_LOCK
	}

	var ol windows.Overlapped

	return windows.LockFileEx(windows.Handle(uintptr(fd)), flags, 0, 1, 0, &ol)
}

func unlockToolSumsFile(fd int) error {
	var ol windows.Overlapped

	return windows.UnlockFileEx(windows.Handle(uintptr(fd)), 0, 1, 0, &ol)
}
