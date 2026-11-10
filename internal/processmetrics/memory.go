// Package processmetrics measures only the current CRM backend process.
package processmetrics

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type MemorySnapshot struct {
	PID               int       `json:"pid"`
	ResidentBytes     uint64    `json:"resident_bytes"`
	PeakResidentBytes uint64    `json:"peak_resident_bytes"`
	SampledAt         time.Time `json:"sampled_at"`
}

func memorySnapshot(resident, peak uint64) (MemorySnapshot, error) {
	if resident == 0 || peak < resident {
		return MemorySnapshot{}, errors.New("invalid process memory counters")
	}
	return MemorySnapshot{PID: os.Getpid(), ResidentBytes: resident, PeakResidentBytes: peak, SampledAt: time.Now().UTC()}, nil
}

func parseProcessStatus(reader io.Reader) (MemorySnapshot, error) {
	values := make(map[string]uint64, 2)
	scanner := bufio.NewScanner(io.LimitReader(reader, 1<<20))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || (fields[0] != "VmRSS:" && fields[0] != "VmHWM:") {
			continue
		}
		if len(fields) != 3 || fields[2] != "kB" {
			return MemorySnapshot{}, errors.New("invalid process memory units")
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil || value > ^uint64(0)/1024 {
			return MemorySnapshot{}, fmt.Errorf("invalid process memory value for %s", fields[0])
		}
		values[fields[0]] = value * 1024
	}
	if err := scanner.Err(); err != nil {
		return MemorySnapshot{}, err
	}
	return memorySnapshot(values["VmRSS:"], values["VmHWM:"])
}
