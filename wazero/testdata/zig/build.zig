const std = @import("std");

pub fn build(b: *std.build.Builder) void {
    const exe = b.addExecutable("zig", "guest.zig");
    exe.wasi_exec_model = .reactor;
    exe.setTarget(b.standardTargetOptions(.{}));
    exe.setBuildMode(.ReleaseFast);
    exe.setOutputDir(".");
    exe.install();
}