build: build_zig build_c

build_zig:
	cd zig && zig build install -Dtarget="wasm32-freestanding" && mv zig.wasm guest.wasm

build_c:
	cd c && emcc guest.c --no-entry -O3 -flto -s WARN_ON_UNDEFINED_SYMBOLS=0 -s STANDALONE_WASM -s WASM=1 -o guest.wasm