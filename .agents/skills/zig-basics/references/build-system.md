# Zig Build System

## When to Use It

Use `zig build-exe`, `zig build-lib`, `zig build-obj`, and `zig test` for simple cases. Use the Zig Build System when:

- Command line becomes too long
- Build process has many steps or targets
- You need concurrency and caching
- You want configuration options for the project
- Build differs based on target system
- You have dependencies on other projects
- You want to avoid external build tools (cmake, make, etc.)
- You want IDE integration

## Simple Executable

hello.zig
```zig
const std = @import("std");

pub fn main() !void {
    std.debug.print("Hello World!\n", .{});
}
```

build.zig
```zig
const std = @import("std");

pub fn build(b: *std.Build) void {
    const exe = b.addExecutable(.{
        .name = "hello",
        .root_module = b.createModule(.{
            .root_source_file = b.path("hello.zig"),
            .target = b.graph.host,
        }),
    });

    b.installArtifact(exe);
}
```

Run with `zig build`. Output goes to `zig-out/bin/hello`.

## Adding a Run Step

```zig
const run_exe = b.addRunArtifact(exe);

const run_step = b.step("run", "Run the application");
run_step.dependOn(&run_exe.step);
```

Run with `zig build run`.

## User-Provided Options

```zig
const windows = b.option(bool, "windows", "Target Microsoft Windows") orelse false;
```

Options become available via `zig build --help` as `-Dwindows=[bool]`.

## Standard Configuration Options

```zig
const target = b.standardTargetOptions(.{});
const optimize = b.standardOptimizeOption(.{});

const exe = b.addExecutable(.{
    .name = "hello",
    .root_module = b.createModule(.{
        .root_source_file = b.path("hello.zig"),
        .target = target,
        .optimize = optimize,
    }),
});
```

Provides `-Dtarget` and `-Doptimize` options.

## Options for Conditional Compilation

```zig
const options = b.addOptions();
options.addOption([]const u8, "version", version);
options.addOption(bool, "have_libfoo", enable_foo);

exe.root_module.addOptions("config", options);
```

In Zig code: `@import("config")`.

## Static Library

```zig
const libfizzbuzz = b.addLibrary(.{
    .name = "fizzbuzz",
    .linkage = .static,
    .root_module = b.createModule(.{
        .root_source_file = b.path("fizzbuzz.zig"),
        .target = target,
        .optimize = optimize,
    }),
});

exe.root_module.linkLibrary(libfizzbuzz);
```

## Dynamic Library

```zig
const libfizzbuzz = b.addLibrary(.{
    .name = "fizzbuzz",
    .linkage = .dynamic,
    .version = .{ .major = 1, .minor = 2, .patch = 3 },
    .root_module = b.createModule(.{
        .root_source_file = b.path("fizzbuzz.zig"),
        .target = target,
        .optimize = optimize,
    }),
});
```

## Testing

```zig
const test_step = b.step("test", "Run unit tests");

const unit_tests = b.addTest(.{
    .root_module = b.createModule(.{
        .root_source_file = b.path("main.zig"),
    }),
});

const run_unit_tests = b.addRunArtifact(unit_tests);
run_unit_tests.skip_foreign_checks = true;
test_step.dependOn(&run_unit_tests.step);
```

Run with `zig build test`.

## Linking System Libraries

```zig
exe.root_module.link_libc = true;
exe.root_module.linkSystemLibrary("z", .{});
```

## Running System Tools

```zig
const tool_run = b.addSystemCommand(&.{"jq"});
tool_run.addArgs(&.{ b.fmt("\\.[\"{s}\"]", .{lang}), "-r" });
tool_run.addFileArg(b.path("words.json"));

const output = tool_run.captureStdOut(.{});
b.getInstallStep().dependOn(&b.addInstallFileWithDir(output, .prefix, "word.txt").step);
```

## Running Project's Tools

```zig
const tool = b.addExecutable(.{
    .name = "word_select",
    .root_module = b.createModule(.{
        .root_source_file = b.path("tools/word_select.zig"),
        .target = b.graph.host,
    }),
});

const tool_step = b.addRunArtifact(tool);
tool_step.addArg("--input-file");
tool_step.addFileArg(b.path("tools/words.json"));
tool_step.addArg("--output-file");
const output = tool_step.addOutputFileArg("word.txt");
```

## Generating Zig Source Code

```zig
const tool = b.addExecutable(.{
    .name = "generate_struct",
    .root_module = b.createModule(.{
        .root_source_file = b.path("tools/generate_struct.zig"),
    }),
});

const tool_step = b.addRunArtifact(tool);
const output = tool_step.addOutputFileArg("person.zig");

exe.root_module.addAnonymousImport("person", .{
    .root_source_file = output,
});
```

## Writing Files

```zig
const wf = b.addWriteFiles();
_ = wf.add("project/version.txt", version);
_ = wf.addCopyFile(source, dest);
```

## Mutating Source Files

```zig
const wf = b.addUpdateSourceFiles();
wf.addCopyFileToSource(generated_file, "src/protocol.zig");

const update_step = b.step("update-protocol", "Update generated files");
update_step.dependOn(&wf.step);
```

## Multi-Target Build

```zig
const targets: []const std.Target.Query = &.{
    .{ .cpu_arch = .aarch64, .os_tag = .macos },
    .{ .cpu_arch = .x86_64, .os_tag = .linux, .abi = .gnu },
    .{ .cpu_arch = .x86_64, .os_tag = .windows },
};

for (targets) |t| {
    const exe = b.addExecutable(.{
        .name = "hello",
        .root_module = b.createModule(.{
            .root_source_file = b.path("hello.zig"),
            .target = b.resolveTargetQuery(t),
            .optimize = .ReleaseSafe,
        }),
    });

    const target_output = b.addInstallArtifact(exe, .{
        .dest_dir = .{ .override = .{ .custom = try t.zigTriple(b.allocator) } },
    });
    b.getInstallStep().dependOn(&target_output.step);
}
```

## Common Commands

```bash
zig build                    # Default install step
zig build run                # Run the application
zig build test               # Run unit tests
zig build -Doptimize=ReleaseSafe
zig build -Dtarget=x86_64-windows
zig build --help             # Show options
zig build -p /custom/path   # Custom install prefix
```
