# USBLauncher - Fyne Edition (Icons Branch)

> A portable Windows executable launcher that scans your USB drive, displays each program with its extracted icon, and lets you search, filter, and launch apps directly from the drive.

## Overview

USBLauncher Fyne Edition is a branch of the main USBLauncher project. The core difference is that this variant uses the [Fyne](https://fyne.io/) GUI framework to render each executable alongside its real Windows icon, extracted directly from the `.exe` file at runtime via the Windows Shell API.

When you run the launcher from a USB drive, it:

1. Detects its own location and treats that directory as the USB root.
2. Performs a two-pass scan — first counting all executables for accurate progress reporting, then loading each one with its icon.
3. Displays the full list in a scrollable Fyne window with a search bar, category filter dropdown, favorites toggle, and a Launch button.
4. Saves your favorites to `Config/favorites.ini` on the drive itself, so they persist between sessions.

Because Fyne requires CGO (a C compiler at build time), the resulting binary is larger and takes longer to build than the plain version, but it provides richer visual identification of programs.

## Relationship to the Main USBLauncher Project

| Feature | Main USBLauncher | This Branch (Fyne Edition) |
|---|---|---|
| GUI framework | Standard Windows GUI / no framework | Fyne v2 |
| Icon display | No | Yes — extracted from each `.exe` |
| Build requirement | Go only | Go + GCC (CGO) |
| Binary size | Small | Larger (~10–20 MB) |
| First-build time | Fast | Several minutes (Fyne compilation) |
| Memory usage | Low | ~50–100 MB |

Use this branch when visual icon identification is important. Use the main branch for minimal footprint or environments where installing a C compiler is not practical.

## Prerequisites

### Go

- Go 1.21 or newer: https://go.dev/dl/

Verify your installation:

```cmd
go version
```

### GCC (C Compiler — required by Fyne/CGO)

Fyne uses CGO to call native graphics libraries. This means a C compiler must be present on the build machine. Choose one of the following options.

**Option A: TDM-GCC (recommended — simplest installer)**

1. Download from: https://jmeubank.github.io/tdm-gcc/
2. Run the installer and select the MinGW-w64 variant.
3. Use the default installation path — the installer adds GCC to `PATH` automatically.

**Option B: MSYS2**

1. Download from: https://www.msys2.org/
2. Install MSYS2, then open the MSYS2 MinGW64 terminal and run:
   ```bash
   pacman -S mingw-w64-x86_64-gcc
   ```
3. Add `C:\msys64\mingw64\bin` to your system `PATH`.

**Option C: MinGW-w64 Standalone**

1. Download from: https://www.mingw-w64.org/
2. Extract the archive and add the `bin` folder to your system `PATH`.

After installing, open a **new** command prompt and verify:

```cmd
gcc --version
```

Expected output (version numbers will vary):

```
gcc (tdm64-1) 10.3.0
```

## Building

All build steps are handled by `build.bat`. Run it from the project directory:

```cmd
build.bat
```

The script will:

1. Verify that `go` is in `PATH`.
2. Verify that `gcc` is in `PATH` and fail with instructions if it is not.
3. Run `go mod tidy` to download all dependencies (Fyne is large; the first run may take several minutes).
4. Compile a **debug** build that keeps a console window open for log output.
5. Compile a **release** build that suppresses the console window entirely.

Two executables are produced:

| File | Purpose |
|---|---|
| `USBLauncher_Fyne.exe` | Release build — no console window |
| `USBLauncher_Fyne_Debug.exe` | Debug build — console window visible for log output |

Note: CGO is enabled automatically by the build script (`set CGO_ENABLED=1`). You do not need to set this manually.

## Installation and Deployment

No installation is required. Copy the built executable(s) to the **root of your USB drive** and run from there.

```
E:\                          <- USB drive root
├── USBLauncher_Fyne.exe     <- launcher executable
├── Config\
│   └── favorites.ini        <- auto-created on first use
├── Games\
│   └── ...
├── Portable\
│   └── ...
└── ...
```

The launcher determines its root by reading its own executable path at startup (`os.Executable()`). All scanning starts from that directory. The `Config\` folder is created automatically the first time favorites are saved.

## Usage

Double-click `USBLauncher_Fyne.exe` (or run `USBLauncher_Fyne_Debug.exe` from a terminal for log output).

The window title shows the detected filesystem type (e.g., `USB Discovery Launcher (exFAT)`).

### Interface

| Element | Function |
|---|---|
| Search bar | Fuzzy-match filter — characters must appear in order but not necessarily consecutively |
| Filter dropdown | Category filter applied on top of the search query |
| "Launch" button | Launches the selected executable in its own directory |
| "Fav" button | Toggles the selected executable as a favorite |
| Scan progress bar | Progress of the directory walk (Pass 1) |
| Load progress bar | Progress of icon extraction and list population (Pass 2) |

### Category Filters

The filter dropdown provides these built-in categories:

| Filter | Match condition |
|---|---|
| All | Every executable |
| Favorites | Only paths marked as favorites |
| Portable | Path contains `portable` (case-insensitive) |
| Games | Path contains `game` |
| Dev | Path contains `programming` or `dev` |
| RE | Path contains `reverse`, `ida`, or `ghidra` |

Filters match against the **full file path**, so folder naming conventions on the drive determine which programs appear under each category.

## Customizing Behavior by Editing the Source

There is no settings panel. All behavioral configuration is done by editing `main.go` and rebuilding with `build.bat`.

### Changing Which Folders Are Skipped During Scanning

In the `startTwoPassScan` function, the following folder names are excluded from the recursive walk:

```go
if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "$") ||
    strings.EqualFold(name, "System Volume Information") ||
    strings.EqualFold(name, "Config") {
    continue
}
```

To exclude additional folders (e.g., a `Drivers` folder you never want to see), add another condition:

```go
strings.EqualFold(name, "Config") ||
strings.EqualFold(name, "Drivers") {
```

### Changing Which File Extensions Are Scanned

Currently only `.exe` files are collected. The relevant check is:

```go
} else if strings.HasSuffix(strings.ToLower(entry.Name()), ".exe") {
    exePaths = append(exePaths, fullPath)
}
```

To also include `.bat` or `.cmd` files, change the condition to:

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

Note: Icon extraction via `ExtractIconExW` only works for PE executables (`.exe`, `.dll`). Non-PE files will fall back to the default blue icon.

### Adding or Modifying Category Filters

Category filter logic lives in the `filterMatch` function:

```go
func (appState *AppState) filterMatch(path string) bool {
    pathLower := strings.ToLower(path)
    switch appState.filterMode {
    case "All":
        return true
    case "★ Favorites":
        return appState.favorites[path]
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

To add a new "Network" category that matches paths containing `network` or `wireshark`:

1. Add the case to `filterMatch`:
   ```go
   case "Network":
       return strings.Contains(pathLower, "network") || strings.Contains(pathLower, "wireshark")
   ```

2. Add the label to the filter dropdown in `createAndRunGUI`:
   ```go
   appState.filterSelect = widget.NewSelect(
       []string{"All", "★ Favorites", "Portable", "Games", "Dev", "RE", "Network"},
       ...
   )
   ```

3. Rebuild with `build.bat`.

### Changing the Window Size

The default window size is set in `createAndRunGUI`:

```go
appState.window.Resize(fyne.NewSize(800, 600))
```

Change the values (width, height in logical pixels) as needed.

### Changing the Icon Size in the List

Each list row renders a 20x20 icon:

```go
icon.SetMinSize(fyne.NewSize(20, 20))
```

Increase this value (e.g., `fyne.NewSize(32, 32)`) for larger icons. Taller rows will result.

### Changing the Icon Preference (Large vs Small)

The icon extraction code prefers the small (16x16) icon. To prefer the large icon instead, swap the priority in `extractIcon`:

```go
// Current: prefers small icon
hIcon := hIconSmall
if hIcon == 0 {
    hIcon = hIconLarge
}

// Change to: prefer large icon
hIcon := hIconLarge
if hIcon == 0 {
    hIcon = hIconSmall
}
```

### Changing the Scan Batch Size

Icons are extracted in batches to allow the list to update progressively. The batch size is:

```go
batchSize := 20
```

Increasing this value reduces UI update frequency but may improve overall throughput on fast drives. Decreasing it makes the list populate more smoothly on slow drives.

## How Icon Extraction Works

Icon extraction is handled entirely by Windows API calls — no third-party icon library is used.

1. `ExtractIconExW` (shell32.dll) — retrieves the large and small `HICON` handles from the target `.exe`.
2. `GetIconInfo` (user32.dll) — extracts the color and mask bitmap handles from the `HICON`.
3. `GetObject` (gdi32.dll) — reads the `BITMAP` struct (width, height, bit depth) from the color bitmap handle.
4. `GetBitmapBits` (gdi32.dll) — copies the raw BGRA pixel data into a Go byte slice.
5. The raw bytes are converted to an `image.RGBA` by reordering channels (BGRA to RGBA) and flipping the Y axis (Windows bitmaps are stored bottom-up).
6. The `image.Image` is stored on the `Executable` struct and passed directly to a `canvas.Image` Fyne widget for rendering.

If any step fails, the default fallback icon is used instead.

## Favorites

Favorites are persisted in plain text at `Config/favorites.ini` relative to the executable's location:

```ini
[Favorites]
E:\Portable\SomeApp\SomeApp.exe=1
E:\Games\GameName\Game.exe=1
```

The file is read at startup and written whenever a favorite is toggled. Full absolute paths are used as keys, so favorites are drive-letter-dependent. If the drive is remounted under a different letter, the paths in `favorites.ini` will need to be updated manually.

## Project Structure

```
USBLauncher/
├── main.go                    # All application logic (single-file project)
├── go.mod                     # Module definition — requires fyne.io/fyne/v2 v2.4.4
├── go.sum                     # Dependency checksums
├── build.bat                  # Windows build script (debug + release)
├── USBLauncher_Fyne.exe       # Pre-built release binary (if present)
└── USBLauncher_Fyne_Debug.exe # Pre-built debug binary (if present)
```

## Performance Expectations

| Phase | Approximate time |
|---|---|
| Directory scan (Pass 1) | Less than 1 second for typical drives |
| Icon extraction (Pass 2) | ~3–5 seconds per 500 executables |
| Memory usage at rest | ~50–100 MB |
| Subsequent builds after first | Fast (seconds) |

Icon extraction is the dominant cost. The two-pass design ensures the list begins populating as soon as Pass 2 starts, without waiting for all icons to load.

## Troubleshooting

**"gcc: command not found" or "CGO_ENABLED=0" errors during build**

GCC is not in `PATH`. Install a C compiler (see Prerequisites), open a new command prompt, and verify with `gcc --version`.

**First build is very slow**

Expected. Fyne and its transitive dependencies are downloaded and compiled from source on the first run. Subsequent builds use the module cache and are significantly faster.

**Icons not showing for some executables**

Some executables have no embedded icon resource. The launcher displays a default blue square in those cases. This is expected behavior.

**Favorites are lost after moving the drive to a different computer**

Favorites use absolute paths including the drive letter. If the drive letter changes, edit `Config/favorites.ini` manually to update the paths, or re-add your favorites once the drive is on the new letter.

**Window appears but the list stays empty**

The scan runs in a background goroutine. Wait for both progress bars to reach 100%. If they never advance, the drive may have no `.exe` files, or the executable may not have been launched from the drive root.
