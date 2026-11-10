package processmetrics

import "os"

func Memory() (MemorySnapshot, error) {
	file, err := os.Open("/proc/self/status")
	if err != nil {
		return MemorySnapshot{}, err
	}
	defer file.Close()
	return parseProcessStatus(file)
}
