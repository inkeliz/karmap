package karmap

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const PAGE_SIZE = 64 * 1024 // 64KB

var (
	_NT                 = windows.NewLazySystemDLL("ntdll.dll")
	_ZwCreateSection    = _NT.NewProc("ZwCreateSection")
	_ZwMapViewOfSection = _NT.NewProc("ZwMapViewOfSection")

	_ZwClose              = _NT.NewProc("ZwClose")
	_ZwUnmapViewOfSection = _NT.NewProc("ZwUnmapViewOfSection")
)

var (
	_SECTION_MAP_READ  = 0x0004
	_SECTION_MAP_WRITE = 0x0002
	_SECTION_QUERY     = 0x0001
)

type sectionHandler struct {
	handler uintptr
	size    uintptr
}

func (s sectionHandler) close() error {
	res, _, _ := _ZwClose.Call(s.handler)
	if res != 0 {
		return fmt.Errorf("zwClose error: %x", res)
	}
	return nil
}

type viewHandler struct {
	handler uintptr
	size    uintptr
}

func (v viewHandler) close() error {
	handle, err := syscall.GetCurrentProcess()
	if err != nil {
		return err
	}

	res, _, _ := _ZwUnmapViewOfSection.Call(uintptr(handle), v.handler)
	if res != 0 {
		return fmt.Errorf("zwUnmapViewOfSection error: %x", res)
	}
	return nil
}

func (v viewHandler) slice() []byte {
	slice := [3]uintptr{uintptr(v.handler), v.size, v.size}
	return *(*[]byte)(unsafe.Pointer(&slice))
}

func newSection(size int) (sectionHandler, error) {
	sh := sectionHandler{size: uintptr(size)}
	siz := uintptr(size)

	res, _, _ := _ZwCreateSection.Call(
		uintptr(unsafe.Pointer(&sh.handler)),
		uintptr(_SECTION_QUERY|_SECTION_MAP_READ|_SECTION_MAP_WRITE),
		0,
		uintptr(unsafe.Pointer(&siz)),
		syscall.PAGE_READWRITE,
		0x8000000,
		0,
	)
	if res != 0 {
		return sh, fmt.Errorf("zwCreateSection error: %x", res)
	}
	return sh, nil
}

func newView(section sectionHandler, size int, pos uintptr, write bool) (viewHandler, error) {
	handle, err := syscall.GetCurrentProcess()
	if err != nil {
		return viewHandler{}, err
	}

	vh := viewHandler{size: uintptr(size), handler: pos}
	sectionOffset := uintptr(0)

	permission := syscall.PAGE_READONLY
	if write {
		permission = syscall.PAGE_READWRITE
	}

	res, _, _ := _ZwMapViewOfSection.Call(
		section.handler,
		uintptr(handle),
		uintptr(unsafe.Pointer(&vh.handler)),
		0,
		vh.size,
		uintptr(unsafe.Pointer(&sectionOffset)),
		uintptr(unsafe.Pointer(&vh.size)),
		0x2,
		0,
		uintptr(permission),
	)
	if res != 0 {
		return vh, fmt.Errorf("zwMapViewOfSection error: %x", res)
	}

	return vh, nil
}
