package karmap

import (
	"errors"
	"sync"
	"unsafe"
)

// MemorySegment is a memory segment.
type MemorySegment struct {
	mutex sync.Mutex

	sectionHandler
	viewHandler

	rc int64
}

// NewMemorySegment creates a new memory segment.
func NewMemorySegment(capacity int) (*MemorySegment, error) {
	return NewMemorySegmentAt(capacity, 0)
}

// NewMemorySegmentAligned creates a new memory segment.
func NewMemorySegmentAligned(capacity int) (*MemorySegment, error) {
	return NewMemorySegmentAt(ToAlignedSize(capacity), 0)
}

// NewMemorySegmentAt creates a new memory segment at the specified position.
//
// The position must be a multiple of 64KB, and the capacity must be a multiple of 64KB,
// and can't overlap with any other memory segment.
func NewMemorySegmentAt(capacity int, pos uintptr) (m *MemorySegment, err error) {
	if (IsAligned(uintptr(capacity)) == false) || (capacity < PAGE_SIZE) {
		return nil, errors.New("capacity must be a multiple of PAGE_SIZE")
	}

	m = &MemorySegment{}
	if m.sectionHandler, err = newSection(capacity); err != nil {
		return m, err
	}
	if m.viewHandler, err = newView(m.sectionHandler, capacity, pos, true); err != nil {
		m.sectionHandler.close()
		return m, err
	}
	return m, nil
}

// Slice returns the underlying byte slice.
// The slice is only valid until the next call to Free,
// it's not resizable, and it's not safe for concurrent use.
//
// Remember that using `append` on a slice returned by Slice
// will cause a memory allocation, and will not extend the
// memory segment.
func (m *MemorySegment) Slice() []byte {
	return m.viewHandler.slice()
}

// Size returns the capacity of the memory segment.
func (m *MemorySegment) Size() int {
	return int(m.viewHandler.size)
}

// Close releases the memory segment.
func (m *MemorySegment) Close() error {
	if m == nil {
		return nil
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.rc > 0 {
		return errors.New("memory segment is still in use")
	}

	if err := m.viewHandler.close(); err != nil {
		return err
	}
	if err := m.sectionHandler.close(); err != nil {
		return err
	}
	return nil
}

// UnsafePointer returns the pointer to the underlying byte slice.
func (m *MemorySegment) UnsafePointer() uintptr {
	return uintptr(unsafe.Pointer(&m.viewHandler.slice()[0]))
}

// AttachView attaches a view to the memory segment.
func (m *MemorySegment) AttachView(pos uintptr) (v View, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	v = View{parent: m}
	if v.viewHandler, err = newView(m.sectionHandler, m.Size(), pos, false); err != nil {
		return v, err
	}

	m.rc++
	return v, nil
}

// View is a view of a memory segment.
type View struct {
	parent *MemorySegment
	viewHandler
}

func (v View) Slice() []byte {
	return v.viewHandler.slice()
}

// Close releases the view.
func (v View) Close() error {
	if v.parent == nil {
		return nil
	}

	v.parent.mutex.Lock()
	defer v.parent.mutex.Unlock()

	v.parent.rc--
	return v.viewHandler.close()
}
