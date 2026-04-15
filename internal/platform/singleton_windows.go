//go:build windows

package platform

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
)

func AcquireSingleton(name string) (release func(), err error) {
	ptr, _ := syscall.UTF16PtrFromString(name)
	r1, _, callErr := procCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(ptr)))
	if r1 == 0 {
		return nil, callErr
	}
	if callErr == syscall.Errno(183) { // ERROR_ALREADY_EXISTS
		syscall.CloseHandle(syscall.Handle(r1))
		return nil, fmt.Errorf("already running")
	}
	handle := syscall.Handle(r1)
	return func() { syscall.CloseHandle(handle) }, nil
}
