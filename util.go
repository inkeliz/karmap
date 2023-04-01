package karmap

// ToAlignedSize returns the size aligned to the page size,
// which is 64KB on Windows and 4KB on Linux.
func ToAlignedSize(size int) int {
	return (size + PAGE_SIZE - 1) & ^(PAGE_SIZE - 1)
}

// ToAlignedSizeWithPadding returns the size aligned to the page size,
// which is 64KB on Windows and 4KB on Linux. That will also add the
// specified margin to the size.
func ToAlignedSizeWithPadding(size int, margin int) int {
	return ToAlignedSize(size) + (margin * PAGE_SIZE)
}

// ToNextAlignedPointer returns the pointer aligned to the page size,
// which is 64KB on Windows and 4KB on Linux.
func ToNextAlignedPointer(ptr uintptr) uintptr {
	return (ptr + PAGE_SIZE - 1) & ^(uintptr(PAGE_SIZE - 1))
}

// IsAligned returns true if the pointer is aligned to the page size.
func IsAligned(ptr uintptr) bool {
	return ptr&^(PAGE_SIZE-1) == ptr
}
