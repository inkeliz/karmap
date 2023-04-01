package wasmkarmap

import (
	"context"
	"fmt"
	"karmap"
	"sync"
	"unsafe"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type Functions struct {
	mutex sync.Mutex

	source *karmap.MemorySegment
	pages  uint32

	views *karmap.View

	segments [2]*karmap.MemorySegment
}

// UnsafeExportView returns the view of the exported memory.
// Should only be used for testing.
func (f *Functions) UnsafeExportView() *karmap.View {
	return f.views
}

func NewFunctionExporter(segment *karmap.MemorySegment, pages uint32) *Functions {
	return &Functions{source: segment, pages: pages}
}

func (f *Functions) CreateView(ctx context.Context, mod api.Module) error {
	res, err := mod.ExportedFunction("karmap_mem_req").Call(ctx, uint64(f.reservedSize()))
	if err != nil {
		return err
	}

	if len(res) != 1 {
		return fmt.Errorf("karmap_mem_req returned %d values, expected 1", len(res))
	}

	if int(res[0]) != f.source.Size() {
		return fmt.Errorf("karmap_mem_req returned %d, expected %d", res[0], f.source.Size())
	}

	return nil
}

func (f *Functions) ExportFunctions(builder wazero.HostModuleBuilder) *Functions {
	builder.NewFunctionBuilder().WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
		f.mutex.Lock()
		defer f.mutex.Unlock()

		if f.views != nil {
			stack = []uint64{uint64(0 | f.source.Size())}
			return
		}

		// internalWazeroMemoryInstance is a private struct in wazero
		type internalWazeroMemoryInstance struct {
			Buffer        []byte
			Min, Cap, Max uint32
			mux           sync.RWMutex
		}

		// wasm.Memory is private, we need to use unsafe
		_hostMemoryInstance := mod.Memory()
		hostMemoryInstance := (*[2]*internalWazeroMemoryInstance)(unsafe.Pointer(&_hostMemoryInstance))[1]

		hostMemoryInstance.mux.Lock()
		defer hostMemoryInstance.mux.Unlock()

		var mem []byte
		var offset, size uint32
		var err error

		// Try to attach the segment 128 times, because there's some race condition
		// We need to allocate a big block of memory, detach it, and then attach it again
		//
		// This is a workaround.
		for i := 0; i < 128; i++ {
			mem, offset, size, err = f.attachSegment(hostMemoryInstance.Buffer, uint32(stack[0]))
			if err == nil {
				break
			}
		}

		if err != nil {
			mod.CloseWithExitCode(ctx, 401)
			return
		}

		// redefine wazero memory (including capacity)
		hostMemoryInstance.Buffer = mem
		hostMemoryInstance.Cap = f.pages

		// pack offset | size
		stack[0] = uint64(offset)<<32 | uint64(size&0xFFFFFFFF)
	}), []api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).Export("karmap_mem_ack")

	return f
}

func (f *Functions) Close() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.views != nil {
		f.views.Close()
	}
	for i := range f.segments {
		f.segments[i].Close()
	}

	return nil
}

func (f *Functions) reservedSize() int {
	return karmap.ToAlignedSizeWithPadding(f.source.Size(), 2)
}

func (f *Functions) attachSegment(memory []byte, ptr uint32) (out []byte, offset uint32, size uint32, err error) {
	guestReserved := ptr
	guestPtr := karmap.ToNextAlignedPointer(uintptr(guestReserved))
	reservedSize := f.reservedSize()
	hostSize := int(f.pages << 16)

	// Find some place to create segment
	tmpSegment, err := karmap.NewMemorySegmentAligned(hostSize)
	if err != nil {
		return nil, 0, 0, err
	}

	tmpPointer := tmpSegment.UnsafePointer()
	if err := tmpSegment.Close(); err != nil {
		return nil, 0, 0, err
	}

	// Create three segments
	seg1cap := int(guestPtr)
	seg1, err := karmap.NewMemorySegmentAt(seg1cap, tmpPointer)
	if err != nil {
		return nil, 0, 0, err
	}
	copy(seg1.Slice(), memory[:seg1cap])

	view, err := f.source.AttachView(tmpPointer + guestPtr)
	if err != nil {
		seg1.Close()
		return nil, 0, 0, err
	}

	seg2cap := hostSize - (seg1cap + reservedSize)
	seg2, err := karmap.NewMemorySegmentAt(seg2cap, tmpPointer+guestPtr+uintptr(reservedSize))
	if err != nil {
		seg1.Close()
		view.Close()
		return nil, 0, 0, err
	}
	copy(seg2.Slice(), memory[seg1cap+reservedSize:])

	f.segments[0] = seg1
	f.segments[1] = seg2
	f.views = &view

	sliceHeader := [3]uintptr{seg1.UnsafePointer(), uintptr(len(memory)), uintptr(hostSize)}
	return *(*[]byte)(unsafe.Pointer(&sliceHeader)), uint32(guestPtr) - guestReserved, uint32(f.source.Size()), nil
}
