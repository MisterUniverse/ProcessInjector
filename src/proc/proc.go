package proc

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type WindowsProc struct {
	ProcessID       int
	ParentProcessID int
	Name            string
	Exe             string
}

var (
	ErrProcessNotFound = errors.New("process not found")
	ErrCreateSnapshot  = errors.New("create snapshot error")
)

func ProcessID(name string) (uint32, error) {
	h, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(h)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	err = windows.Process32First(h, &entry)
	if err != nil {
		return 0, err
	}

	for {
		if windows.UTF16ToString(entry.ExeFile[:]) == name {
			return entry.ProcessID, nil
		}

		err = windows.Process32Next(h, &entry)
		if err != nil {
			if err == windows.ERROR_NO_MORE_FILES {
				return 0, fmt.Errorf("%q not found", name)
			}
			return 0, err
		}
	}
}

func Processes() ([]WindowsProc, error) {
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(handle)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	err = windows.Process32First(handle, &entry)
	if err != nil {
		return nil, err
	}

	var results []WindowsProc
	for {
		results = append(results, WindowsProc{
			ProcessID:       int(entry.ProcessID),
			ParentProcessID: int(entry.ParentProcessID),
			Exe:             windows.UTF16ToString(entry.ExeFile[:]),
		})

		err = windows.Process32Next(handle, &entry)
		if err != nil {
			if err == windows.ERROR_NO_MORE_FILES {
				break
			}
			return nil, err
		}
	}

	return results, nil
}

func FindProcessByName(processes []WindowsProc, name string) int {
	for _, p := range processes {
		if strings.ToLower(p.Exe) == strings.ToLower(name) {
			return p.ProcessID
		}
	}
	return 0
}

type winProc windows.Handle

// returns a pointer to a process handle
func OpenProcessHandle(processID int) (winProc, error) {
	const PROCESS_ALL_ACCESS = 0x1F0FFF
	handle, err := windows.OpenProcess(PROCESS_ALL_ACCESS, false, uint32(processID))
	if err != nil {
		return 0, err
	}
	return winProc(handle), nil
}
