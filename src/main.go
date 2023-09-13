package main

import (
	"ProcessInjector/src/proc"
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	VirtualAllocMemCommit  = 0x00001000
	VirtualAllocMemReserve = 0x00002000
	PageReadWrite          = 0x04
	SizeOfChar             = 4
)

var (
	kernel32Module         = windows.NewLazySystemDLL("kernel32.dll")
	procGetModuleHandle    = kernel32Module.NewProc("GetModuleHandleW")
	procVirtualAllocEx     = kernel32Module.NewProc("VirtualAllocEx")
	procCreateRemoteThread = kernel32Module.NewProc("CreateRemoteThread")
)

/*
Original Function:
LPVOID VirtualAllocEx(

	[in]           HANDLE hProcess,
	[in, optional] LPVOID lpAddress,
	[in]           SIZE_T dwSize,
	[in]           DWORD  flAllocationType,
	[in]           DWORD  flProtect

);
*/
func VirtualAllocEx(hProcess windows.Handle, lpAddress uintptr, dwSize uint, flAllocationType, flProtect uint32) uintptr {
	ret, _, _ := procVirtualAllocEx.Call(
		uintptr(hProcess),
		lpAddress,
		uintptr(dwSize),
		uintptr(flAllocationType),
		uintptr(flProtect),
	)

	return ret
}

/*
Original Function:
HANDLE CreateRemoteThread(

	[in]  HANDLE                 hProcess,
	[in]  LPSECURITY_ATTRIBUTES  lpThreadAttributes,
	[in]  SIZE_T                 dwStackSize,
	[in]  LPTHREAD_START_ROUTINE lpStartAddress,
	[in]  LPVOID                 lpParameter,
	[in]  DWORD                  dwCreationFlags,
	[out] LPDWORD                lpThreadId

);
*/
func CreateRemoteThread(hProcess windows.Handle, lpThreadAttributes uintptr, dwStackSize uint, lpStartAddress, lpParameter uintptr, dwCreationFlags uint32, lpThreadId *uintptr) uintptr {
	ret, _, _ := procCreateRemoteThread.Call(
		uintptr(hProcess),
		lpThreadAttributes,
		uintptr(dwStackSize),
		lpStartAddress,
		lpParameter,
		uintptr(dwCreationFlags),
		uintptr(unsafe.Pointer(lpThreadId)),
	)

	return ret
}

func main() {
	// Gets processid returns a unint32
	a, err := proc.ProcessID(os.Args[0])
	if err != nil {
		fmt.Println(err)
		return
	}

	// returns a ptr to a process handle
	// ptr := proc.OpenProcessHandle(int(a))
	// fmt.Println(ptr)

	// // returns a list of processes
	// list, err := proc.Processes()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	//proc.FindProcessByName(list, os.Args[1])

	// finds a process in the list by name and injects code
	err = InjectDLL(a, os.Args[1])
	if err != nil {
		fmt.Println("Error injecting DLL:", err)
	}
}

func InjectDLL(pid uint32, dll string) error {
	if f, err := os.Stat(dll); err != nil || f.IsDir() {
		return errors.New("invalid DLL file")
	}

	processHandle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|
		windows.PROCESS_CREATE_THREAD|
		windows.PROCESS_VM_OPERATION|
		windows.PROCESS_VM_WRITE|
		windows.PROCESS_VM_READ, false, pid)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(processHandle)

	kernel32, err := windows.LoadLibrary("kernel32.dll")
	if err != nil {
		return err
	}
	defer windows.FreeLibrary(kernel32)

	loadLibraryA, err := windows.GetProcAddress(kernel32, "LoadLibraryA")
	if err != nil {
		return err
	}

	size := (uint)((len(dll) + 1) * SizeOfChar)
	memAddress := VirtualAllocEx(processHandle, uintptr(0), size, VirtualAllocMemCommit|VirtualAllocMemReserve, windows.PAGE_READWRITE)
	ptr, err := syscall.BytePtrFromString(dll)
	if err != nil {
		return err
	}

	var bytesWritten uintptr
	err = windows.WriteProcessMemory(processHandle, memAddress, ptr, uintptr(size), &bytesWritten)
	if err != nil {
		return err
	}
	fmt.Println("Bytes Written: ", bytesWritten)

	var threadID uintptr
	CreateRemoteThread(processHandle, uintptr(0), 0, loadLibraryA, memAddress, 0, &threadID)
	if err != nil {
		return err
	}
	fmt.Println("Created thread ID: ", threadID)

	return nil
}
