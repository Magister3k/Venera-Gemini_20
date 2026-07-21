package utils

import "syscall"

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsole = kernel32.NewProc("GetConsoleWindow")
	procShowWindow = user32.NewProc("ShowWindow")
)

const (
	SW_HIDE = 0
	SW_SHOW = 5
)

// HideConsole скрывает окно консоли текущего процесса в Windows
func HideConsole() {
	hwnd, _, _ := procGetConsole.Call()
	if hwnd != 0 {
		_, _, _ = procShowWindow.Call(hwnd, uintptr(SW_HIDE))
	}
}

// ShowConsole отображает окно консоли текущего процесса в Windows
func ShowConsole() {
	hwnd, _, _ := procGetConsole.Call()
	if hwnd != 0 {
		_, _, _ = procShowWindow.Call(hwnd, uintptr(SW_SHOW))
	}
}
