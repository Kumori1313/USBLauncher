# USB Launcher - HTA Edition (For Restricted PCs)

A portable USB executable launcher that works on restricted PCs where compiled executables may be blocked.

## Why HTA?

HTA (HTML Application) files:

* Run using `mshta.exe`, a **built-in Windows component**
* Are often **allowed by corporate policies** that block unknown executables
* Require **no installation** - just copy and run
* Work on **all Windows versions** from XP to 11

## Features

* Scans USB drive for executables
* Fuzzy search
* Category filters (All, Favorites, Portable, Games, Dev, RE)
* Favorites system with persistent storage
* Clean, modern-looking interface
* No icons (limitation of HTA, but keeps it simple and fast)

## Usage

1. Copy `USBLauncher.hta` to the root of your USB drive
2. Double-click to run
3. Wait for the scan to complete
4. Double-click any executable to launch, or select and click "Launch"

## Limitations vs Go Version

|Feature|Go Version|HTA Version|
|-|-|-|
|Startup speed|Fast|Moderate|
|Icons|No|No|
|Works on restricted PCs|Usually blocked|Usually works|
|Binary size|~8MB|~15KB|
|Requires runtime|No|No (mshta built-in)|

## Troubleshooting

### "Windows cannot run this application"

* Rarely, mshta.exe itself may be blocked
* Try running from a local drive first to test

### Slow scanning

* HTA uses VBScript which is slower than Go
* Large drives with many files will take longer

### Security warnings

* Some antivirus may flag HTA files
* This is a false positive - the code is plaintext and can be inspected

### Favorites not saving

* Ensure the USB drive is not write-protected
* Check if `Config` folder can be created

## Customization

The HTA file is plain HTML/CSS/VBScript/JavaScript. You can edit it with any text editor to:

* Change colors (CSS in `<style>` section)
* Add filter categories (modify `filterMatch` function)
* Adjust window size (in `init()` function)

## Security Note

HTA files run with full user permissions - the same as any application you'd run. This launcher only:

* Reads file system to find executables
* Writes to `Config\\\\\\\\favorites.ini` for favorites
* Launches executables you explicitly select

The code is fully visible and auditable - just open the file in Notepad.

