package main

import (
	"syscall"
	"unsafe"
)

type ttySize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func getTTYSize() ttySize {
	ts := ttySize{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ts)))

	if int(retCode) == -1 {
		panic(errno)
	}

	return ts
}
