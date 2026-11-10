//go:build windows

package recovery

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	processQueryLimitedInformation = 0x1000
	stillActive                    = 259
)

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess = kernel32.NewProc("OpenProcess")
	procGetExitCode = kernel32.NewProc("GetExitCodeProcess")
	procCloseHandle = kernel32.NewProc("CloseHandle")
)

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, _, _ := procOpenProcess.Call(uintptr(processQueryLimitedInformation), 0, uintptr(pid))
	if h == 0 {
		return false
	}
	defer procCloseHandle.Call(h)

	var exitCode uint32
	r, _, _ := procGetExitCode.Call(h, uintptr(unsafe.Pointer(&exitCode)))
	if r == 0 {
		return false
	}
	return exitCode == stillActive
}

func terminateProcess(pid int) error {
	if pid <= 0 {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
