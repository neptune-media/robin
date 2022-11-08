package ffmpeg

import (
	"golang.org/x/sys/windows"
	"os"
)

func setLowerPriority(p *os.Process) error {
	// Acquire a handle to the child process
	// PROCESS_SET_INFORMATION is the access level needed when calling SetPriorityClass
	// https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-setpriorityclass
	h, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION, false, uint32(p.Pid))
	if err != nil {
		return err
	}

	// Attempt to modify the priority class of the child process
	return windows.SetPriorityClass(h, windows.BELOW_NORMAL_PRIORITY_CLASS)
}
