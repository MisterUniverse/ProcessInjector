package proc

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"Ysera/src/win32"

	"golang.org/x/sys/windows"
)

type WindowsProc struct {
	ProcessID       int
	ParentProcessID int
	Name            string
	Exe             string
}

const TH32CS_SNAPPROCESS = 0x00000002

type Process struct {
	ProcessID int
	Name      string
	ExePath   string
}

type winProc uintptr

var (
	ErrProcessNotFound = errors.New("process not found")
	ErrCreateSnapshot  = errors.New("create snapshot error")
	ErrAlreadyInjected = errors.New("dll already injected")
	ErrModuleNotExits  = errors.New("can't found module")
	ErrModuleSnapshot  = errors.New("create module snapshot failed")
)

func ProcessID(name string) (uint32, error) {
	h, e := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if e != nil {
		return 0, e
	}

	p := windows.ProcessEntry32{Size: 568}

	for {
		e := windows.Process32Next(h, &p)
		if e != nil {
			return 0, e
		}

		if windows.UTF16ToString(p.ExeFile[:]) == name {
			return p.ProcessID, nil
		}
	}
	return 0, fmt.Errorf("%q not found", name)
}

func Processes() ([]WindowsProc, error) {
	handle, err := windows.CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(handle)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	// get the first process
	err = windows.Process32First(handle, &entry)
	if err != nil {
		return nil, err
	}

	results := make([]WindowsProc, 0, 50)
	for {
		results = append(results, newWindowsProcess(&entry))

		err = windows.Process32Next(handle, &entry)
		if err != nil {
			// windows sends ERROR_NO_MORE_FILES on last process
			if err == syscall.ERROR_NO_MORE_FILES {
				return results, nil
			}
			return nil, err
		}
	}
}

func FindProcessByName(processes []WindowsProc, name string) int {
	for _, p := range processes {
		if strings.ToLower(p.Exe) == strings.ToLower(name) {
			return p.ProcessID
		}
	}
	return 0
}

func newWindowsProcess(e *windows.ProcessEntry32) WindowsProc {
	// Find when the string ends for decoding
	end := 0
	for {
		if e.ExeFile[end] == 0 {
			break
		}
		end++
	}

	return WindowsProc{
		ProcessID:       int(e.ProcessID),
		ParentProcessID: int(e.ParentProcessID),
		Exe:             syscall.UTF16ToString(e.ExeFile[:end]),
	}
}

// FindProcessByName get process information by name
func GetProcessInfoByName(name string) (*Process, error) {
	handle, _ := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if handle == 0 {
		return nil, ErrCreateSnapshot
	}
	defer syscall.CloseHandle(handle)

	var procEntry = syscall.ProcessEntry32{}
	procEntry.Size = uint32(unsafe.Sizeof(procEntry))
	var process Process

	for true {
		if nil != syscall.Process32Next(handle, &procEntry) {
			break
		}

		_exeFile := win32.UTF16PtrToString(&procEntry.ExeFile[0])
		if name == _exeFile {
			process.Name = _exeFile
			process.ProcessID = int(procEntry.ProcessID)

			process.ExePath = _exeFile
			return &process, nil
		}

	}
	return nil, ErrProcessNotFound
}

// returns a pointer to a process handle
func OpenProcessHandle(processId int) winProc {
	const PROCESS_ALL_ACCESS = 0x1F0FFF

	kernel32 := syscall.MustLoadDLL("kernel32.dll")
	proc := kernel32.MustFindProc("OpenProcess")
	handle, _, _ := proc.Call(ptr(PROCESS_ALL_ACCESS), ptr(true), ptr(processId))
	return winProc(handle)
}

func ptr(val interface{}) uintptr {
	switch val.(type) {
	case string:
		return uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(val.(string))))
	case int:
		return uintptr(val.(int))
	default:
		return uintptr(0)
	}
}
