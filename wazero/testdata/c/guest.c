#include <emscripten.h>
#include <stdint.h>
#include <stdlib.h>

/* READ ZIG SOURCE CODE TO UNDERSTAND THIS FILE */

// Reseved memory for input, which mmap will be mapped to,
// in some area of reserved_input space.
uint8_t* reserved_input_memory;
uint32_t reserved_input_size;

// Memory that will be used for input, which is a part of reserved_input_memory,
// and it's mmaped.
uint8_t* input_memory;
uint32_t input_size;

__attribute__((export_name("_start")))
void _start() {}

extern uint64_t  karmap_mem_ack(uint32_t ptr);

__attribute__((export_name("karmap_mem_req")))
EMSCRIPTEN_KEEPALIVE uint32_t karmap_mem_req(uint32_t size) {
    if (reserved_input_memory == NULL) {
        reserved_input_memory = malloc(size);
        reserved_input_size = size;
    }
    
    uint64_t offsetsize = karmap_mem_ack((uint32_t)reserved_input_memory);
    
    input_memory = reserved_input_memory;
    input_memory += offsetsize >> 32;
    input_size = offsetsize & 0xFFFFFFFF;
    
    return input_size;
}

__attribute__((export_name("work")))
EMSCRIPTEN_KEEPALIVE uint32_t work() {
    uint32_t i = 0;
    uint32_t acc = 0;

    uint32_t * input_memory_as_u32 = (uint32_t*)input_memory;
    uint32_t input_size_as_u32 = input_size / 4;
    while(i < input_size_as_u32) {
        acc += input_memory_as_u32[i];
        i++;
    }
    return acc;
}

// Only used for debugging/testing/bechmarking
__attribute__((export_name("debug_test_malloc")))
EMSCRIPTEN_KEEPALIVE uint32_t debug_test_malloc(uint32_t size) {
    input_memory = malloc(size);
    return (uint32_t)input_memory;
}

// Only used for debugging/testing/bechmarking
__attribute__((export_name("debug_try_write")))
EMSCRIPTEN_KEEPALIVE void debug_try_write() {
    input_memory[0] = 0xFF;
    input_memory[input_size - 1] = 42;
}