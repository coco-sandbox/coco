const std = @import("std");

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    const exe = b.addExecutable(.{
        .name = "cocovisor",
        .root_source_file = b.path("src/main.zig"),
        .target = target,
        .optimize = optimize,
    });

    // Add vmm and api modules
    const vmm_module = b.addModule("vmm", .{
        .root_source_file = b.path("src/vmm.zig"),
    });
    const api_module = b.addModule("api", .{
        .root_source_file = b.path("src/api.zig"),
    });

    exe.root_module.addImport("vmm", vmm_module);
    exe.root_module.addImport("api", api_module);
    api_module.addImport("vmm", vmm_module);

    b.installArtifact(exe);

    const run_cmd = b.addRunArtifact(exe);
    if (b.args) |args| {
        run_cmd.addArgs(args);
    }

    const run_step = b.step("run", "Run the app");
    run_step.dependOn(&run_cmd.step);

    const unit_tests = b.addTest(.{
        .root_source_file = b.path("src/main.zig"),
        .target = target,
        .optimize = optimize,
    });

    const run_unit_tests = b.addRunArtifact(unit_tests);
    const test_step = b.step("test", "Run unit tests");
    test_step.dependOn(&run_unit_tests.step);
}
