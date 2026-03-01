# USB Launcher - HTA Edition (Silent / Restricted PCs)

> A single-file portable launcher for USB drives that scans for executables and launches them without triggering UAC prompts — built as an HTA (HTML Application) so it runs via the built-in `mshta.exe` host on any Windows machine.

---

## Overview

USB Launcher HTA Edition is a self-contained `.hta` file you drop on a USB drive. When opened, it recursively scans the drive for `.exe` files, presents them in a searchable and categorised list, and launches the one you select — all without needing a compiler, runtime, or installer.

It is a variant of the main USBLauncher project (which is a compiled Go binary). The key difference is that this version trades the Go toolchain for the Windows-native `mshta.exe` host, making it usable on locked-down or corporate machines where running an unknown `.exe` is blocked by policy.

**What it does:**

- Scans the USB drive root recursively for `.exe` files
- Displays results in a resizable, searchable list
- Supports category filtering (All, Favorites, Portable, Games, Dev, RE)
- Persists a favorites list to `Config\favorites.ini` on the drive
- Detects and displays the drive's filesystem type (NTFS, FAT32, exFAT)
- Launches selected executables with their own folder as the working directory

---

## Prerequisites

| Requirement | Details |
|---|---|
| Windows version | Windows XP through Windows 11 |
| Runtime | `mshta.exe` (built into every Windows installation) |
| Permissions | Standard user account — no elevation required |
| Tools to edit | Any plain-text editor (Notepad, VS Code, Notepad++, etc.) |

No installation, no compilation, no dependencies.

---

## Usage

### Running the launcher

1. Copy `USBLauncher.hta` to the **root** of your USB drive.
2. Double-click `USBLauncher.hta`.
3. Windows opens it with `mshta.exe` automatically.
4. Wait a few seconds for the scan to complete.
5. Use the search box or category filter to find an executable.
6. **Double-click** a row, or select a row and click **Launch**, to run it.

You can also run it explicitly from the command line:

```bat
mshta.exe E:\USBLauncher.hta
```

Replace `E:\` with the actual drive letter of your USB drive.

### UI reference

| Control | Function |
|---|---|
| Search box | Live substring filter on executable filename |
| Category dropdown | Filter by category (see [Category filters](#category-filters) below) |
| **Fav** button | Toggle the selected executable as a favorite |
| **Launch** button | Run the selected executable |
| Double-click a row | Run that executable immediately |

### Favorites

Select any executable and click **Fav** to mark it as a favorite. Favorites are stored in `Config\favorites.ini` on the USB drive. Use the **Favorites** option in the dropdown to show only starred entries.

---

## How It Works

### HTA and the UAC bypass

An `.hta` file is an HTML Application — a standard Windows feature since IE 5 (1999). Windows executes it using `mshta.exe`, a signed Microsoft binary that ships with every Windows installation.

When you double-click an `.hta` file, the Windows shell launches `mshta.exe` (not your file directly), so:

- No UAC prompt appears for the HTA itself — `mshta.exe` is a trusted system binary.
- Policies that block unknown executables do not apply to `mshta.exe`.
- The application runs with the **same permissions as the current user** — it does not escalate privilege.

When the HTA in turn launches a selected `.exe`, it uses `Shell.Application.ShellExecute` with the `"open"` verb. This means the launched program inherits the user's permissions. If the launched executable requests elevation, Windows will show a UAC prompt for that program specifically — that behaviour is controlled by the target executable, not by this launcher.

### Scanning

The VBScript `ScanAllExes` function performs a breadth-first traversal of the drive using `Scripting.FileSystemObject`. It skips the following folders by name:

- Any folder whose name begins with `.` or `$`
- `System Volume Information`
- `Config` (the launcher's own config folder)
- `Recycler`
- `$Recycle.Bin`

All `.exe` files found are collected into a pipe-delimited string and returned to JavaScript in a single call, which then parses and renders the list.

---

## Customising the Launcher

All configuration is done by editing `USBLauncher.hta` directly in a text editor. There is no settings UI. The sections below pinpoint the exact places to change.

### Window size

In the `init()` JavaScript function (approximately line 338):

```javascript
function init() {
    window.resizeTo(850, 650);  // Change width and height here
    ...
}
```

Adjust the two numbers to change the initial window dimensions in pixels.

### Category filters

The dropdown and its matching logic are in two places.

**Step 1 — Add an `<option>` to the dropdown** (approximately line 157):

```html
<select id="filterSelect" onchange="applyFilter()">
    <option value="All">All</option>
    <option value="Favorites">★ Favorites</option>
    <option value="Portable">Portable</option>
    <option value="Games">Games</option>
    <option value="Dev">Dev</option>
    <option value="RE">RE</option>
    <!-- Add a new line here, e.g.: -->
    <option value="Utils">Utils</option>
</select>
```

The `value` attribute is what the filter logic receives; the visible label can be anything.

**Step 2 — Add a matching case to `filterMatch`** (approximately line 437):

```javascript
function filterMatch(path, mode) {
    var pathLower = path.toLowerCase();
    switch (mode) {
        case "All":       return true;
        case "Favorites": return IsFavorite(path);
        case "Games":     return pathLower.indexOf("game") >= 0;
        case "Dev":       return pathLower.indexOf("programming") >= 0
                              || pathLower.indexOf("dev") >= 0;
        case "RE":        return pathLower.indexOf("d:\\apps\\re\\") >= 0;
        case "Portable":  return pathLower.indexOf("portable") >= 0;
        // Add your new case here, e.g.:
        case "Utils":     return pathLower.indexOf("utils") >= 0;
    }
    return true;
}
```

The match logic uses `indexOf` on the **full path** (lowercased), so you can match on folder names, drive paths, or any substring of the path.

> **Note:** The built-in "RE" category matches a hardcoded drive path (`d:\apps\re\`). If your drive letter or folder layout differs, update that case to match your own path.

### Excluded folders (scan exclusions)

In the VBScript `ScanAllExes` function (approximately line 286):

```vbscript
If Left(fn, 1) <> "." And Left(fn, 1) <> "$" And _
   LCase(fn) <> "system volume information" And _
   LCase(fn) <> "config" And LCase(fn) <> "recycler" And _
   LCase(fn) <> "$recycle.bin" Then
```

To skip additional folders, add more `And LCase(fn) <> "your-folder-name"` clauses. Comparisons are case-insensitive.

Example — also skip a folder named `drivers`:

```vbscript
If Left(fn, 1) <> "." And Left(fn, 1) <> "$" And _
   LCase(fn) <> "system volume information" And _
   LCase(fn) <> "config" And LCase(fn) <> "recycler" And _
   LCase(fn) <> "$recycle.bin" And LCase(fn) <> "drivers" Then
```

### File extension filter

The scanner currently finds only `.exe` files. This is set in `ScanAllExes` (approximately line 301):

```vbscript
If LCase(fso.GetExtensionName(file.Name)) = "exe" Then
```

To also include `.com` files, change the condition to:

```vbscript
Dim ext : ext = LCase(fso.GetExtensionName(file.Name))
If ext = "exe" Or ext = "com" Then
```

### Config folder location

The favorites file path is set in `InitPaths` (approximately line 194):

```vbscript
configDir = usbRoot & "\Config"
favFile   = configDir & "\favorites.ini"
```

Change `"\Config"` to any folder name you prefer, and change `"\favorites.ini"` to rename the file.

### Appearance (colours, fonts, spacing)

All visual styling is in the `<style>` block near the top of the file. It is standard CSS. Key selectors:

| Selector | Controls |
|---|---|
| `body` | Background colour, padding |
| `.list-item.selected` | Highlight colour for the selected row |
| `button.primary` | Launch button colour |
| `.search-box:focus` | Search box focus ring colour |
| `.list-item .path` | Path text colour and truncation |

---

## File and Folder Layout on the USB Drive

After first run, the drive will contain:

```
E:\
├── USBLauncher.hta       <- The launcher (place here before first run)
└── Config\
    └── favorites.ini     <- Created automatically; stores starred executables
```

The `Config` folder is created automatically on first run if it does not exist. The launcher uses the folder containing `USBLauncher.hta` as the USB root, so the file can technically live anywhere on the drive — but placing it at the root ensures the scan covers the whole drive.

---

## HTA Edition vs Go Binary Edition

| Feature | Go Binary | HTA Edition |
|---|---|---|
| Requires build toolchain | Yes (Go) | No |
| File size | ~8 MB | ~15 KB |
| Startup / scan speed | Fast | Moderate (VBScript) |
| Works on restricted PCs | Often blocked | Usually works (mshta.exe is trusted) |
| Shell icons for executables | No | No |
| Requires runtime | No (self-contained) | No (mshta.exe is built in) |
| Editable without recompiling | No | Yes (plain text) |
| Windows version support | Windows 7+ | Windows XP+ |

Use the HTA edition when:
- The target machine's policy blocks unsigned or unknown `.exe` files.
- You want to customise the tool without installing Go or recompiling.
- You need the absolute smallest file to carry on a drive.

Use the Go binary edition when:
- Scan speed on large drives is a priority.
- You are on a machine with no policy restrictions and prefer a compiled binary.

---

## Troubleshooting

### "Windows cannot open this file" or nothing happens on double-click

`mshta.exe` may have been disabled or unregistered. Test by running directly:

```bat
mshta.exe E:\USBLauncher.hta
```

If that also fails, `mshta.exe` may have been removed or blocked by group policy on that machine. There is no workaround in that case short of administrator intervention.

### Antivirus flags the HTA file

Some antivirus products generically flag all `.hta` files because the format can be misused. The file is plain text — open it in Notepad to audit the full source before running it. No external network requests are made; no data leaves the machine.

### The scan is slow

VBScript's `FileSystemObject` is significantly slower than native Go I/O. On a drive with tens of thousands of files the scan may take several seconds. The status bar at the bottom shows elapsed time once the scan finishes.

### Favorites are not saved after closing

The `Config` folder or `favorites.ini` could not be written. Common causes:

- The USB drive is **write-protected** (physical switch on the drive, or a Windows policy).
- The drive is formatted as a **read-only filesystem**.

Check that you can create a text file on the drive manually. If the drive has a write-protect switch, slide it to the unlocked position.

### A specific folder is being scanned but should not be

Add the folder name to the exclusion list in `ScanAllExes` — see [Excluded folders](#excluded-folders-scan-exclusions) above.

### Category filter shows no results

Check the `filterMatch` switch statement. The `RE` category matches a hardcoded path (`d:\apps\re\`). If your drive letter is different, that case will never match. Update it to match your own path structure.

---

## Security Considerations

The launcher runs with the permissions of the user who opened it — no elevation is requested or granted. It performs three operations only:

1. Reads the filesystem to enumerate `.exe` files.
2. Reads and writes `Config\favorites.ini` to persist favorites.
3. Calls `ShellExecute` on an executable you explicitly select.

The source code is fully visible in the `.hta` file itself. Open it in any text editor to inspect or audit everything it does before running it.
