package main

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"

	"github.com/AllenDang/w32"
)

/*
The primary difference between the standard DLL injection example and the manual map injection example
is the method of loading the DLL. In the standard injection, the LoadLibrary function is called within
the target process, relying on the operating system to handle the DLL loading process automatically.
In contrast, the manual map injection manually loads the DLL into the injector's memory and maps it
into the target process's memory space.
*/
func main() {
	// Define the path to the DLL you want to inject
	dllPath := "C:\\path\\to\\your.dll"

	// Define the process name or ID of the target process
	targetProcessName := "target.exe"

	// Find the target process
	pid, err := findProcessID(targetProcessName)
	if err != nil {
		log.Fatalf("Failed to find process: %v", err)
	}

	// Open the target process with required permissions
	processHandle, err := w32.OpenProcess(w32.PROCESS_CREATE_THREAD|w32.PROCESS_VM_OPERATION|w32.PROCESS_VM_WRITE|w32.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		log.Fatalf("Failed to open process: %v", err)
	}
	defer w32.CloseHandle(processHandle)

	// Load the DLL into the injector's memory
	dllHandle := w32.LoadLibrary(dllPath)
	if dllHandle == 0 {
		log.Fatalf("Failed to load DLL: %v", syscall.GetLastError())
	}
	defer w32.FreeLibrary(dllHandle)

	// Get the address of the DLL's entry point
	dllEntryPoint := w32.GetProcAddress(dllHandle, "DllMain")
	if dllEntryPoint == 0 {
		log.Fatalf("Failed to get DLL entry point: %v", syscall.GetLastError())
	}

	// Allocate memory in the target process for the DLL path
	dllPathAddr, err := w32.VirtualAllocEx(processHandle, 0, len(dllPath), w32.MEM_COMMIT, w32.PAGE_READWRITE)
	if err != nil {
		log.Fatalf("Failed to allocate memory in target process: %v", err)
	}

	// Write the DLL path to the target process
	err = w32.WriteProcessMemory(processHandle, dllPathAddr, []byte(dllPath), uint(len(dllPath)))
	if err != nil {
		log.Fatalf("Failed to write DLL path to target process: %v", err)
	}

	// Create a remote thread in the target process to execute the DLL's entry point
	threadHandle, err := w32.CreateRemoteThread(processHandle, nil, 0, dllEntryPoint, dllPathAddr, 0, nil)
	if err != nil {
		log.Fatalf("Failed to create remote thread: %v", err)
	}
	defer w32.CloseHandle(threadHandle)

	fmt.Println("DLL injected successfully!")
}

// Helper function to find the process ID by name
func findProcessID(processName string) (int, error) {
	handle, err := w32.CreateToolhelp32Snapshot(w32.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer w32.CloseHandle(handle)

	var pe32 w32.PROCESSENTRY32
	pe32.DwSize = uint32(unsafe.Sizeof(pe32))

	if err := w32.Process32First(handle, &pe32); err != nil {
		return 0, err
	}

	for {
		if syscall.UTF16ToString(pe32.SzExeFile[:]) == processName {
			return int(pe32.Th32ProcessID), nil
		}

		if err := w32.Process32Next(handle, &pe32); err != nil {
			break
		}
	}

	return 0, fmt.Errorf("process not found")
}
