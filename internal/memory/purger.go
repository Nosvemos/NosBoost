package memory

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// SystemMemoryListInformation is the undocumented NT class for system memory commands.
	SystemMemoryListInformation = 80

	// SYSTEM_MEMORY_LIST_COMMAND enums
	MemoryPurgeStandbyList   = 4
	MemoryFlushModifiedList  = 5

	// Privileges required for memory list manipulation
	SeProfileSingleProcessPrivilege = "SeProfileSingleProcessPrivilege"
	SeIncreaseQuotaPrivilege        = "SeIncreaseQuotaPrivilege"
)

var (
	ntdll                      = windows.NewLazySystemDLL("ntdll.dll")
	procNtSetSystemInformation = ntdll.NewProc("NtSetSystemInformation")
)

// SetTokenPrivilege programmatically enables or disables specific security privileges in our current token.
func SetTokenPrivilege(privilegeName string, enable bool) error {
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token)
	if err != nil {
		return fmt.Errorf("failed to open current process token: %w", err)
	}
	defer token.Close()

	var luid windows.LUID
	err = windows.LookupPrivilegeValue(nil, windows.StringToUTF16Ptr(privilegeName), &luid)
	if err != nil {
		return fmt.Errorf("failed to lookup privilege value %s: %w", privilegeName, err)
	}

	var tp windows.Tokenprivileges
	tp.PrivilegeCount = 1
	tp.Privileges[0].Luid = luid
	if enable {
		tp.Privileges[0].Attributes = windows.SE_PRIVILEGE_ENABLED
	} else {
		tp.Privileges[0].Attributes = 0
	}

	// Adjust privileges
	err = windows.AdjustTokenPrivileges(token, false, &tp, 0, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to adjust token privileges: %w", err)
	}

	// Check if AdjustTokenPrivileges succeeded partially (GetLastError can return ERROR_NOT_ALL_ASSIGNED)
	if err := windows.GetLastError(); err != nil {
		if errors.Is(err, windows.ERROR_NOT_ALL_ASSIGNED) {
			return fmt.Errorf("privilege %s not assigned to current token", privilegeName)
		}
	}

	return nil
}

// ntSetSystemInformation executes the low-level undocumented NT call inside a safe recover block.
func ntSetSystemInformation(class uint32, info unsafe.Pointer, length uint32) (uintptr, error) {
	var status uintptr
	var err error

	// Safe recover block to catch any kernel access violation panics
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("recovered from kernel access violation panic: %v", r)
			}
		}()

		r1, _, e1 := procNtSetSystemInformation.Call(
			uintptr(class),
			uintptr(info),
			uintptr(length),
		)
		status = r1
		if status != 0 {
			// e1 contains system-level error or e1 is empty
			if e1 != nil && e1.Error() != "The operation completed successfully." {
				err = e1
			} else {
				err = fmt.Errorf("NTSTATUS error code: 0x%X", status)
			}
		}
	}()

	return status, err
}

// PurgeStandbyList flushes the entire Windows file cache and standby list.
func PurgeStandbyList() error {
	// 1. Enable SeProfileSingleProcessPrivilege
	if err := SetTokenPrivilege(SeProfileSingleProcessPrivilege, true); err != nil {
		return fmt.Errorf("failed to enable SeProfileSingleProcessPrivilege: %w", err)
	}

	command := uint32(MemoryPurgeStandbyList)
	_, err := ntSetSystemInformation(SystemMemoryListInformation, unsafe.Pointer(&command), uint32(unsafe.Sizeof(command)))
	if err != nil {
		return fmt.Errorf("NtSetSystemInformation MemoryPurgeStandbyList failed: %w", err)
	}

	return nil
}

// FlushModifiedList flushes modified memory pages back to storage, clearing high-speed memory buffers.
func FlushModifiedList() error {
	// 1. Enable SeProfileSingleProcessPrivilege
	if err := SetTokenPrivilege(SeProfileSingleProcessPrivilege, true); err != nil {
		return fmt.Errorf("failed to enable SeProfileSingleProcessPrivilege: %w", err)
	}

	command := uint32(MemoryFlushModifiedList)
	_, err := ntSetSystemInformation(SystemMemoryListInformation, unsafe.Pointer(&command), uint32(unsafe.Sizeof(command)))
	if err != nil {
		return fmt.Errorf("NtSetSystemInformation MemoryFlushModifiedList failed: %w", err)
	}

	return nil
}
