# USB Launcher - Fyne Edition (With Icons)

A fast USB executable launcher with icon support, built using the Fyne GUI toolkit.

## Features

* Two-pass scanning with accurate progress
* **Icon extraction** from executables
* Fuzzy search
* Category filters (All, Favorites, Portable, Games, Dev, RE)
* Favorites system
* Native cross-platform GUI

## Prerequisites

Fyne requires **CGO** which means you need a C compiler installed.

### Step 1: Install a C Compiler

Choose ONE of these options:

**Option A: TDM-GCC (Recommended - Easiest)**

1. Download from: https://jmeubank.github.io/tdm-gcc/
2. Run the installer, select "MinGW-w64" version
3. Use default installation path
4. It automatically adds itself to PATH

**Option B: MSYS2**

1. Download from: https://www.msys2.org/
2. Install and run MSYS2
3. Run: `pacman -S mingw-w64-x86\\\\\\\_64-gcc`
4. Add `C:\\\\\\\\msys64\\\\\\\\mingw64\\\\\\\\bin` to your PATH

**Option C: MinGW-w64 Standalone**

1. Download from: https://www.mingw-w64.org/
2. Extract and add `bin` folder to PATH

### Step 2: Verify GCC Installation

Open a NEW command prompt and run:

```cmd
gcc --version
```

You should see something like:

```
gcc (tdm64-1) 10.3.0
```

### Step 3: Build the Launcher

```cmd
cd usblauncher\\\\\\\_fyne
build.bat
```

The first build will take several minutes as Fyne downloads and compiles its dependencies.

## Troubleshooting

### "gcc: command not found"

* GCC is not in your PATH
* Restart your command prompt after installing GCC
* Verify the GCC bin folder is in your PATH

### Build takes forever

* Normal for first build (Fyne is large)
* Subsequent builds are much faster

### "CGO\_ENABLED=0" errors

* Ensure you have GCC installed
* The build script sets CGO\_ENABLED=1

### Icons not showing

* Some executables don't have icons
* The launcher uses a default blue icon as fallback

## Performance

With icons enabled, expect:

* Scan phase: ~0.5s
* Load phase: ~3-5s for 500 executables (icon extraction)
* Memory: ~50-100MB

This is slower than the no-icon version but provides better visual identification.

