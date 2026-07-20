package metrics

import (
	"syscall"
	"unsafe"
)

var (
	psapi                    = syscall.NewLazyDLL("psapi.dll")
	procGetProcessMemoryInfo = psapi.NewProc("GetProcessMemoryInfo")
)

const (
	processQueryInformation = 0x0400
	processVMRead           = 0x0010
)

type processMemoryCounters struct {
	cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uint64 // в Windows x64 SIZE_T это 64 бита
	WorkingSetSize             uint64
	QuotaPeakPagedPoolUsage    uint64
	QuotaPagedPoolUsage        uint64
	QuotaPeakNonPagedPoolUsage uint64
	QuotaNonPagedPoolUsage     uint64
	PagefileUsage              uint64
	PeakPagefileUsage          uint64
}

// GetProcessRAM возвращает использование RAM (WorkingSetSize) процессом по его PID (в байтах)
// Потребление RAM без сторонних библиотек.
func GetProcessRAM(pid int) (uint64, error) {
	handle, err := syscall.OpenProcess(processQueryInformation|processVMRead, false, uint32(pid))
	if err != nil {
		return 0, err
	}
	defer syscall.CloseHandle(handle)

	var counters processMemoryCounters
	counters.cb = uint32(unsafe.Sizeof(counters))

	ret, _, err := procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&counters)),
		uintptr(counters.cb),
	)

	if ret == 0 {
		return 0, err
	}

	return counters.WorkingSetSize, nil
}
