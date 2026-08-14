package metrics

import (
	"syscall"
	"unsafe"
)

var (
	psapi              = syscall.NewLazyDLL("psapi.dll")
	procGetProcMemInfo = psapi.NewProc("GetProcessMemoryInfo")
)

const (
	procQueryInfo = 0x0400
	procVMRead    = 0x0010
)

type procMemCounters struct {
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

// GetProcRAM возвращает использование RAM процессом по его PID (в байтах)
func GetProcRAM(pid int) (uint64, error) {
	handle, err := syscall.OpenProcess(procQueryInfo|procVMRead, false, uint32(pid))
	if err != nil {
		return 0, err
	}
	defer syscall.CloseHandle(handle)

	var counters procMemCounters
	counters.cb = uint32(unsafe.Sizeof(counters))

	ret, _, err := procGetProcMemInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&counters)),
		uintptr(counters.cb),
	)

	if ret == 0 {
		return 0, err
	}

	return counters.WorkingSetSize, nil
}
