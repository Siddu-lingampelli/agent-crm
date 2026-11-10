package processmetrics

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestParseProcessStatus(t *testing.T) {
	got, err := parseProcessStatus(strings.NewReader("Name: crm\nVmSize: 100000 kB\nVmHWM: 2048 kB\nVmRSS: 1024 kB\nRssAnon: 800 kB\n"))
	if err != nil || got.ResidentBytes != 1024*1024 || got.PeakResidentBytes != 2048*1024 || got.PID != os.Getpid() || got.SampledAt.IsZero() {
		t.Fatalf("process memory: %+v %v", got, err)
	}
	for _, input := range []string{
		"MemTotal: 16000 kB\nMemAvailable: 6000 kB\n",
		"VmRSS: 1024 kB\n", "VmRSS: 0 kB\nVmHWM: 2048 kB\n",
		"VmRSS: 2048 kB\nVmHWM: 1024 kB\n",
		"VmRSS: invalid kB\nVmHWM: 2048 kB\n",
		"VmRSS: 1024 B\nVmHWM: 2048 kB\n",
		"VmRSS: 18446744073709551615 kB\nVmHWM: 2048 kB\n",
	} {
		if _, err := parseProcessStatus(strings.NewReader(input)); err == nil {
			t.Fatalf("accepted invalid process status: %q", input)
		}
	}
}

func TestLiveMemory(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		t.Skip("process collector is supported on Windows and Linux")
	}
	got, err := Memory()
	if err != nil {
		t.Fatal(err)
	}
	if got.PID != os.Getpid() || got.ResidentBytes == 0 || got.PeakResidentBytes < got.ResidentBytes || got.SampledAt.IsZero() {
		t.Fatalf("invalid process measurements: %+v", got)
	}
}
