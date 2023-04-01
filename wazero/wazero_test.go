package wazero

import (
	"context"
	_ "embed"
	"karmap"
	"karmap/wazero/wasmkarmap"
	"runtime/debug"
	"testing"
	"unsafe"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/assemblyscript"
	"github.com/tetratelabs/wazero/imports/emscripten"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

var (
	//go:embed testdata/c/guest.wasm
	_guestC []byte

	//go:embed testdata/zig/guest.wasm
	_guestZIG []byte
)

func Test_Wazero(t *testing.T) {
	// data is the source of truth for the memory,
	// all guests will read that same memory.
	data, err := karmap.NewMemorySegmentAligned(100_000 * 4)
	if err != nil {
		t.Error(err)
		return
	}

	config := wazero.NewRuntimeConfigCompiler().WithMemoryLimitPages(256).WithMemoryCapacityFromMax(true) // 16MB
	ctx := context.Background()

	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	defer runtime.Close(nil)

	env := runtime.NewHostModuleBuilder("env")
	wasi_snapshot_preview1.NewFunctionExporter().ExportFunctions(env)
	emscripten.NewFunctionExporter().ExportFunctions(env)
	assemblyscript.NewFunctionExporter().WithAbortMessageDisabled().ExportFunctions(env)

	// Create memory view
	view := wasmkarmap.NewFunctionExporter(data, 256).ExportFunctions(env)
	defer view.Close()

	exports, err := env.Instantiate(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	defer exports.Close(context.Background())

	wasm, err := runtime.Instantiate(ctx, _guestZIG)
	if err != nil {
		t.Error(err)
		return
	}
	defer wasm.Close(context.Background())

	// Request the guest to allocate memory and then we mmap it.
	if err = view.CreateView(ctx, wasm); err != nil {
		t.Error(err)
		return
	}

	// Write that into the shared-memory.
	dataSlice := data.Slice()
	dataU32 := [3]uintptr{uintptr(unsafe.Pointer(&dataSlice[0])), uintptr(len(dataSlice) / 4), uintptr(cap(dataSlice) / 4)}

	dataX := *(*[]uint32)(unsafe.Pointer(&dataU32))
	sum := uint32(0)
	for i := 0; i < len(dataX); i++ {
		dataX[i] = uint32(i)
		sum += uint32(i)
	}

	// Call the guest function, which will read the shared-memory.
	res, err := wasm.ExportedFunction("work").Call(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	if res[0] != uint64(sum) {
		t.Error("result not equal", res[0], sum)
		return
	}
}

func initWebAssemblyModule(data *karmap.MemorySegment, pageCapacity uint32, guest []byte) (wasm api.Module, close func(), err error) {
	config := wazero.NewRuntimeConfigCompiler().WithMemoryLimitPages(pageCapacity).WithMemoryCapacityFromMax(true) // 16MB
	ctx := context.Background()

	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	defer func() {
		if err != nil {
			runtime.Close(nil)
		}
	}()

	env := runtime.NewHostModuleBuilder("env")
	wasi_snapshot_preview1.NewFunctionExporter().ExportFunctions(env)
	emscripten.NewFunctionExporter().ExportFunctions(env)
	assemblyscript.NewFunctionExporter().WithAbortMessageDisabled().ExportFunctions(env)

	view := wasmkarmap.NewFunctionExporter(data, pageCapacity).ExportFunctions(env)
	defer func() {
		if err != nil {
			view.Close()
		}
	}()

	if _, err = env.Instantiate(ctx); err != nil {
		return wasm, nil, err
	}

	wasm, err = runtime.Instantiate(ctx, guest)
	if err != nil {
		return wasm, nil, err
	}

	if err = view.CreateView(ctx, wasm); err != nil {
		return wasm, nil, err
	}

	return wasm, func() {
		wasm.Close(ctx)
		view.Close()
		runtime.Close(nil)
	}, err
}

func Test_NativeTryWrite(t *testing.T) {
	data, err := karmap.NewMemorySegmentAligned(100_000 * 4)
	if err != nil {
		t.Error(err)
		return
	}

	view, err := data.AttachView(0)
	if err != nil {
		t.Error(err)
		return
	}

	debug.SetPanicOnFault(true)

	defer func() {
		if r := recover(); r != nil {

		} else {
			t.Error("should panic")
		}
	}()

	// This will cause a panic
	view.Slice()[0] = 0x01
}

func Test_WazeroGuestTryWrite(t *testing.T) {
	data, err := karmap.NewMemorySegmentAligned(100_000 * 4)
	if err != nil {
		t.Error(err)
		return
	}

	wasm, closes, err := initWebAssemblyModule(data, 255, _guestZIG)
	if err != nil {
		t.Error(err)
		return
	}
	defer closes()

	debug.SetPanicOnFault(true)

	defer func() {
		if r := recover(); r == nil {
			t.Error("should panic")
		}
	}()

	_, err = wasm.ExportedFunction("debug_try_write").Call(nil)
	if err != nil {
		t.Error(err)
		return
	}

}

func Test_WazeroMultipleGuests(t *testing.T) {

}
