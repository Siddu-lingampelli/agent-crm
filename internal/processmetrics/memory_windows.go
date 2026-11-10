package processmetrics

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var getProcessMemoryInfo = windows.NewLazySystemDLL("kernel32.dll").NewProc("K32GetProcessMemoryInfo")

// Layout matches PROCESS_MEMORY_COUNTERS_EX, including pointer-sized SIZE_T fields.
// https://learn.microsoft.com/en-us/windows/win32/api/psapi/ns-psapi-process_memory_counters_ex
type processMemoryCounters struct {
	Size                  uint32
	PageFaultCount        uint32
	PeakWorkingSetSize    uintptr
	WorkingSetSize        uintptr
	QuotaPeakPagedPool    uintptr
	QuotaPagedPool        uintptr
	QuotaPeakNonPagedPool uintptr
	QuotaNonPagedPool     uintptr
	PagefileUsage         uintptr
	PeakPagefileUsage     uintptr
	PrivateUsage          uintptr
}

// Memory reads the current process's working set, including native allocations.
func Memory() (MemorySnapshot, error) {
	if err := getProcessMemoryInfo.Find(); err != nil {
		return MemorySnapshot{}, err
	}
	var counters processMemoryCounters
	counters.Size = uint32(unsafe.Sizeof(counters))
	result, _, err := getProcessMemoryInfo.Call(uintptr(windows.CurrentProcess()), uintptr(unsafe.Pointer(&counters)), uintptr(counters.Size))
	if result == 0 {
		return MemorySnapshot{}, fmt.Errorf("GetProcessMemoryInfo failed: %w", err)
	}
	return memorySnapshot(uint64(counters.WorkingSetSize), uint64(counters.PeakWorkingSetSize))
}
