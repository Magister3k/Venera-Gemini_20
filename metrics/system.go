package metrics

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetDiskFreeSpaceExW  = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// memoryStatusEx структура для получения данных об оперативной памяти (Windows API)
type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

// GetSystemRAM собирает статистику по оперативной памяти (п.6.1 ТЗ) без сторонних библиотек.
// Возвращает свободную память в байтах и процент свободной памяти.
func GetSystemRAM() (uint64, float64, error) {
	var memInfo memoryStatusEx
	memInfo.dwLength = uint32(unsafe.Sizeof(memInfo))

	ret, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memInfo)))
	if ret == 0 {
		return 0, 0, err
	}

	freeBytes := memInfo.ullAvailPhys
	// dwMemoryLoad содержит процент ИСПОЛЬЗОВАННОЙ памяти (0..100)
	freePercent := 100.0 - float64(memInfo.dwMemoryLoad)

	return freeBytes, freePercent, nil
}

// GetDiskSpace собирает статистику свободного места на диске (п.6.1 ТЗ) без сторонних библиотек.
// path - путь (буква диска "C:\" или папка).
// Возвращает свободное место в байтах и процент.
func GetDiskSpace(path string) (uint64, float64, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}

	var freeBytesAvailableToCaller uint64
	var totalNumberOfBytes uint64
	var totalNumberOfFreeBytes uint64

	ret, _, err := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailableToCaller)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)

	if ret == 0 {
		return 0, 0, err
	}

	if totalNumberOfBytes == 0 {
		return 0, 0, nil
	}

	freePercent := (float64(freeBytesAvailableToCaller) / float64(totalNumberOfBytes)) * 100.0

	return freeBytesAvailableToCaller, freePercent, nil
}
