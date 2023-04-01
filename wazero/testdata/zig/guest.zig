const std = @import("std");

// reserved_input_memory is used to "reserve" memory for input_memory,
// in some GC-languages, that is mandatory to avoid been lost by GC.
//
// The "reserved" is larger than input_memory, because mmap requires
// the size of the memory to be a multiple of the page size, so
// reserved_input_memory can be 128KB larger than required. The
// mmap will happen inside reserved_input_memory.
//
// In Zig, it is not necessary, but it is a good practice to do so,
// to make clear that such memory is reserved for input_memory.
var reserved_input_memory: []u8 = undefined;

// input_memory is the memory that is used to store the input data.
//
// It is a slice of reserved_input_memory, which is mmaped.
var input_memory: []u8 = undefined;

pub fn main() void {}

// karmap_mem_ack is called by the guest, passing the pointer to the
// reserved_input_memory. The host will then mmap the memory region
// (or part of it) and inform the guest about the offset and size,
// in that order, packed in a u64.
extern "env" fn karmap_mem_ack(ptr: u32) u64;

// karmap_mem_req is called by the host, forcing the guest to create a memory
// region that will be used to store the input data.
//
// Then, the guest will call karmap_mem_ack, passing the pointer to the
// host. The host will then mmap the memory region (or part of it) and
// inform the guest about the size and offset (based on reserved_input_memory).
export fn karmap_mem_req(size: u32) u32 {
    if (reserved_input_memory.len == 0) {
        reserved_input_memory = std.heap.page_allocator.alloc(u8, size) catch unreachable;
    }

    var offsetsize: u64 = karmap_mem_ack(@ptrToInt(reserved_input_memory.ptr));

    input_memory = reserved_input_memory[@intCast(usize, offsetsize >> 32)..@intCast(usize, ((offsetsize >> 32) + (offsetsize & 0xFFFFFFFF)))];

    return @intCast(u32, offsetsize);
}

export fn work() u32 {
    var acc: u32 = 0;

    var slice: [2]usize = [2]usize{@ptrToInt(input_memory.ptr), input_memory.len / 4};
    var slice_32: []u32 = @ptrCast(*[]u32, &slice).*;
    for (slice_32) |value| {
        acc += value;
    }

    return acc;
}

// Only used for debugging/testing/bechmarking
export fn debug_test_malloc(size: u32) u32 {
    input_memory = std.heap.page_allocator.alloc(u8, size) catch unreachable;
    return @ptrToInt(input_memory.ptr);
}

// Only used for debugging/testing/bechmarking
export fn debug_try_write() void {
    input_memory[0] = 0xFF;
    input_memory[input_memory.len - 1] = 42;
}