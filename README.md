KARMAP
----

Karmap is a simple way to share a single memory segment between Host and WebAssembly modules, 
using Wazero as the WebAssembly runtime.

## IT'S NOT READY YET - DO NOT USE

## Motivation

The main reason for this project is to provide a way to share memory between Host and WebAssembly modules,
without need to copy data between them. This is useful when you want to share a single memory segment between
multiples modules, or when you want to share a memory segment between a Host and a WebAssembly module.

## Security

That project heavily uses unsafe, including changing Wazero internals, and creating segments of memory
using OS specific APIs.

That is not stable and not fully tested. **It is not recommended to use it in production.**

## How it works

```go
    // Error handling omitted for brevity
	
    // Create your primary memory segment (will be shared with the guest)
	data, _ := karmap.NewMemorySegmentAligned(100_000 * 4)
	
	// Define the memory limit (in pages)
	memoryLimit := 256

	// Create the Wazero runtime (WithMemoryLimitPages(memoryLimit) is required)
	config := wazero.NewRuntimeConfigCompiler().WithMemoryLimitPages(memoryLimit).WithMemoryCapacityFromMax(true) // 16MB
	ctx := context.Background()

	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	defer runtime.Close(nil)

    
	env := runtime.NewHostModuleBuilder("env")

    // Define the Host module (export functions to the guest), define the memory segment and
	// the maximum size of the memory segment.
	view := wasmkarmap.NewFunctionExporter(data, memoryLimit).ExportFunctions(env)
	defer view.Close()

	// Instantiate the Host module
	exports, _ := env.Instantiate(ctx)
	defer exports.Close(context.Background())

	// Instantiate the Guest module
	wasm, _ := runtime.Instantiate(ctx, YOUR_WASM_MODULE)
	defer wasm.Close(context.Background())

	// Request the guest to allocate memory and then we mmap it.
	if err = view.CreateView(ctx, wasm); err != nil {
		t.Error(err)
		return
	}
	
	// Now we have `data` shared between Host and Guest modules. 🥳
```

## Limitations

- Only support one memory view
  - Just one segment of the memory can be shared between Host and Guest modules, for now.

- Only works on Windows (for now)
  - It should be easy to port to other platforms, but I don't have the time to do it.

- Only works on Wazero as the WebAssembly runtime
  - Technically, it should work on any WebAssembly runtime.

- Only read-only memory segments
  - The Guest module can only read the memory segment, it can't write to it, for now.

- Requires some export/imported WebAssembly (small "ABI") 
  - Two functions need to be implemented by the Host and Guest modules.

- Requires fixed memory size 
  - WASM can still call "grow", but the host should never reallocate the memory

- Requires 128KB of overhead and 64KB pages minimum size.
    - If you need 1KB, it will require reserve 192KB, that is not an issue on large memory segments.

