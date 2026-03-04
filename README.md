# USBLauncher

> A portable Windows GUI tool that scans a USB drive for executables and presents them in a searchable, filterable launch list.

## Overview

USBLauncher is a self-contained Windows application written in Go. Place it at the root of a USB drive and run it — it will recursively walk every subfolder, collect all `.exe` files it finds, and display them in a native Win32 list box. From there you can fuzzy-search the list, filter by category, mark favorites, and launch any program with a double-click or the Launch button.

The tool uses a **two-pass scan**: the first pass counts executables and reports progress; the second pass loads them in batches so the list populates incrementally while the scan is still running. Scanning and UI updates happen concurrently and the interface stays responsive throughout.

There is no GUI settings panel. All behavioral customization — file extensions, excluded folders, category filter keywords — is done by editing the source code directly and rebuilding.

## Prerequisites

| Requirement | Notes |
|---|---|
| **Windows** (any modern version) | The tool uses `shell32.dll`, `kernel32.dll`, and the Win32 common-controls subsystem — it is Windows-only |
| **Go 1.24 or later** | Required only if you want to build from source; pre-built binaries need no Go installation |
| **GCC / MinGW-w64** | Required at build time because `github.com/lxn/walk` uses CGo — install via [MSYS2](https://www.msys2.org/) (`pacman -S mingw-w64-x86_64-gcc`) and ensure the MinGW `bin` directory is on your `PATH` |

> If you only want to run the tool, copy `USBLauncher.exe` and `USBLauncher.exe.manifest` to the USB drive root. No Go or GCC installation is needed on the target machine.

## Building from Source

Clone or copy the repository, then run the provided build script from a command prompt inside the project directory.

```bat
build.bat
```

The script performs three steps in order:

1. Verifies that `go` is on `PATH`
2. Runs `go mod tidy` to download dependencies
3. Compiles two binaries:
   - `USBLauncher_Debug.exe` — keeps the console window open; shows live diagnostic output and is useful during development
   - `USBLauncher.exe` — release build compiled with `-H windowsgui -s -w`; no console window, smaller binary

Both binaries are created in the project root alongside their `.manifest` files.

To build manually without the script:

```bat
REM Debug build
go build -o USBLauncher_Debug.exe .

REM Release build
go build -ldflags="-H windowsgui -s -w" -o USBLauncher.exe .
```

## Deploying to a USB Drive

Copy these four files to the root of your USB drive:

```
USBLauncher.exe
USBLauncher.exe.manifest
USBLauncher_Debug.exe          (optional — only if you want debug output)
USBLauncher_Debug.exe.manifest (required if you include the debug binary)
```

The `.manifest` files must always sit next to their corresponding `.exe`. They instruct Windows to load the modern Common Controls v6 visual style; without them the list box and other controls will not render correctly.

When launched, the application creates a `Config/` folder at the drive root (next to the `.exe`) and stores `favorites.ini` there.

## Usage

Double-click `USBLauncher.exe` from the USB drive root.

The window title shows the detected filesystem type (e.g., `USB Discovery Launcher (NTFS)`). A warning is shown for FAT32 (4 GB file limit) or exFAT (symlink limitations).

| Control | Action |
|---|---|
| **Search box** | Fuzzy-filters the list as you type — characters must appear in order but need not be contiguous |
| **Category dropdown** | Filters by category (see [Category Filters](#modifying-category-filters) below) |
| **★ Fav button** | Toggles the selected executable as a favorite; favorites are saved to `Config/favorites.ini` |
| **Launch button** | Launches the selected executable via `ShellExecuteW` with the executable's own directory as the working directory |
| **Double-click** | Same as the Launch button |

### Progress Indicators

Two progress bars appear during startup:

- **Scanning executables** — tracks directory traversal progress during pass 1
- **Loading launcher** — tracks how many executables have been added to the list during pass 2

Both bars reach 100% when the scan is complete. The status label at the bottom reports the total count once loading finishes.

## Customization (Source Code Edits)

Because there is no settings UI, all customization requires editing `main.go` and rebuilding.

---

### Changing the File Extension Filter

The scanner currently matches only `.exe` files. The relevant line is in `startTwoPassScan()` inside `main.go`:

```go
// main.go — around line 368
} else if strings.HasSuffix(strings.ToLower(entry.Name()), ".exe") {
    exePaths = append(exePaths, fullPath)
}
```

To also include `.bat` and `.cmd` files, replace that condition with a helper check:

```go
} else {
    lower := strings.ToLower(entry.Name())
    if strings.HasSuffix(lower, ".exe") ||
        strings.HasSuffix(lower, ".bat") ||
        strings.HasSuffix(lower, ".cmd") {
        exePaths = append(exePaths, fullPath)
    }
}
```

---

### Adding or Removing Excluded Folders

By default the scanner skips directories whose names match any of the following rules (in `startTwoPassScan()`):

```go
// main.go — around lines 362–365
if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "$") ||
    strings.EqualFold(name, "System Volume Information") ||
    strings.EqualFold(name, "Config") {
    continue
}
```

| Rule | What it skips |
|---|---|
| Starts with `.` | Hidden Unix-style directories (`.git`, `.vscode`, etc.) |
| Starts with `$` | Windows system directories (`$RECYCLE.BIN`, `$SysReset`, etc.) |
| `System Volume Information` | Windows restore-point metadata folder |
| `Config` | The launcher's own configuration directory |

To exclude additional folders, add more `strings.EqualFold` comparisons to that block:

```go
strings.EqualFold(name, "Config") ||
strings.EqualFold(name, "Windows") ||
strings.EqualFold(name, "Drivers") {
```

To stop skipping a built-in folder (for example, if you want `.dot` directories scanned), remove the corresponding condition.

---

### Modifying Category Filters

The category dropdown is populated in `createAndRunGUI()`:

```go
// main.go — around line 261
app.filterCombo.SetModel([]string{"All", "★ Favorites", "Portable", "Games", "Dev", "RE"})
```

The matching logic for each category lives in `filterMatch()`:

```go
// main.go — around lines 513–530
func (app *AppState) filterMatch(path string) bool {
    pathLower := strings.ToLower(path)
    switch app.filterMode {
    case "All":
        return true
    case "★ Favorites":
        return app.favorites[path]
    case "Games":
        return strings.Contains(pathLower, "game")
    case "Dev":
        return strings.Contains(pathLower, "programming") || strings.Contains(pathLower, "dev")
    case "RE":
        return strings.Contains(pathLower, "reverse") || strings.Contains(pathLower, "ida") || strings.Contains(pathLower, "ghidra")
    case "Portable":
        return strings.Contains(pathLower, "portable")
    }
    return true
}
```

Each `case` tests whether a substring appears anywhere in the full file path. To add a new category:

1. Add the display name to the `SetModel` slice.
2. Add a matching `case` in `filterMatch()`.

Example — adding a "Security" category that matches paths containing `security`, `nmap`, or `wireshark`:

```go
// In SetModel:
[]string{"All", "★ Favorites", "Portable", "Games", "Dev", "RE", "Security"}

// In filterMatch():
case "Security":
    return strings.Contains(pathLower, "security") ||
        strings.Contains(pathLower, "nmap") ||
        strings.Contains(pathLower, "wireshark")
```

To rename an existing category, change the string in both the `SetModel` call and the corresponding `case` label so they match exactly.

---

### Changing the Window Size

Window dimensions are set in `createAndRunGUI()`:

```go
// main.go — around line 224
app.mainWindow.SetSize(walk.Size{Width: 750, Height: 550})
```

Adjust `Width` and `Height` to any pixel values you prefer.

---

### Changing the Window Title

The title is assembled from the detected filesystem type:

```go
// main.go — around line 222
title := fmt.Sprintf("USB Discovery Launcher (%s)", app.fsType)
```

Replace the format string to use any static or dynamic title text you want.

## Project Structure

```
USBLauncher/
├── main.go                         # All application logic — scan, GUI, filter, launch
├── go.mod                          # Module definition; requires go 1.24+
├── go.sum                          # Dependency checksums
├── build.bat                       # Build script — produces Debug and Release binaries
├── USBLauncher.exe.manifest        # Win32 manifest for the release binary (Common Controls v6)
└── USBLauncher_Debug.exe.manifest  # Win32 manifest for the debug binary
```

The `Config/` directory is created at runtime on the USB drive root and is not part of the source tree:

```
Config/
└── favorites.ini    # Auto-generated; stores starred executables in INI format
```

## Manifest Files Explained

Windows application manifests are XML sidecar files that the OS loader reads before starting an executable. The two manifests in this project both declare a dependency on **Microsoft.Windows.Common-Controls v6**, which enables the modern visual style for native controls (the list box, progress bars, buttons, etc.). Without the manifest the controls fall back to the Windows Classic look.

The manifest does **not** request elevated UAC privileges — the tool runs as a standard user. If a launched executable itself requires elevation, Windows will prompt the user via the normal UAC consent dialog at the time of launch.

The manifest file must be named `<executable-name>.manifest` and must sit in the same directory as the `.exe` it applies to. If you rename the binary, rename the manifest to match.

## Dependencies

| Package | Version | Purpose |
|---|---|---|
| `github.com/lxn/walk` | `v0.0.0-20210112085537` | Win32 GUI toolkit (windows, layout, controls) |
| `github.com/lxn/win` | `v0.0.0-20210218163916` | Low-level Win32 API bindings (`SW_SHOWNORMAL`, etc.) |
| `golang.org/x/sys` | `v0.40.0` | Indirect — required by `lxn/walk` |
| `gopkg.in/Knetic/govaluate.v3` | `v3.0.0` | Indirect — required by `lxn/walk` |

All dependencies are resolved automatically by `go mod tidy` or during the `build.bat` run.