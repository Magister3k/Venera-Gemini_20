package metrics

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGlobalMemStatusEx    = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetDiskFreeSpaceExW  = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// memStatusEx структура для получения данных об оперативной памяти (Windows API)
type memStatusEx struct {
	dwLen                   uint32
	dwMemLoad               uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtVirtual      uint64
}

// GetSysRAM собирает статистику по оперативной памяти.
// Возвращает свободную память в байтах и процент свободной памяти.
func GetSysRAM() (uint64, float64, error) {
	var memInfo memStatusEx
	memInfo.dwLen = uint32(unsafe.Sizeof(memInfo))

	ret, _, err := procGlobalMemStatusEx.Call(uintptr(unsafe.Pointer(&memInfo)))
	if ret == 0 {
		return 0, 0, err
	}

	freeBytes := memInfo.ullAvailPhys
	// dwMemLoad содержит процент использованной памяти (0..100)
	freePerc := 100.0 - float64(memInfo.dwMemLoad)

	return freeBytes, freePerc, nil
}

// GetDiskSpace собирает статистику свободного места на диске без сторонних библиотек.
// path - путь (буква диска "C:\" или папка).
// Возвращает свободное место в байтах и процент.
func GetDiskSpace(path string) (uint64, float64, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}

	var freeBytesAvailToCaller uint64
	var totalNumberOfBytes uint64
	var totalNumberOfFreeBytes uint64

	ret, _, err := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailToCaller)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)

	if ret == 0 {
		return 0, 0, err
	}

	if totalNumberOfBytes == 0 {
		return 0, 0, nil
	}

	freePerc := (float64(freeBytesAvailToCaller) / float64(totalNumberOfBytes)) * 100.0

	return freeBytesAvailToCaller, freePerc, nil
}
