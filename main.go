// USB Discovery Launcher - Fyne Edition (With Icons)
// Features: Two-pass scan, icon extraction, fuzzy search, favorites

package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Executable represents a found .exe file
type Executable struct {
	Name string
	Path string
	Icon image.Image
}

// AppState holds all application state
type AppState struct {
	execList     []*Executable
	filteredList []*Executable
	favorites    map[string]bool
	filterMode   string
	searchQuery  string
	usbRoot      string
	configDir    string
	favFile      string
	fsType       string

	// Scan state
	totalExeCount int
	loadedCount   int

	// GUI elements
	window       fyne.Window
	list         *widget.List
	searchEntry  *widget.Entry
	filterSelect *widget.Select
	scanProgress *widget.ProgressBar
	loadProgress *widget.ProgressBar
	statusLabel  *widget.Label

	// Default icon
	defaultIcon image.Image

	// Selection tracking
	selectedID int
	hasSelection bool

	// Synchronization
	mutex sync.RWMutex
}

// Windows API structures and functions
var (
	shell32              = syscall.NewLazyDLL("shell32.dll")
	user32               = syscall.NewLazyDLL("user32.dll")
	gdi32                = syscall.NewLazyDLL("gdi32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procExtractIconExW   = shell32.NewProc("ExtractIconExW")
	procGetIconInfo      = user32.NewProc("GetIconInfo")
	procGetDIBits        = gdi32.NewProc("GetDIBits")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC         = gdi32.NewProc("DeleteDC")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procDestroyIcon      = user32.NewProc("DestroyIcon")
	procGetBitmapBits    = gdi32.NewProc("GetBitmapBits")
	procGetObject        = gdi32.NewProc("GetObjectW")
	procGetVolumeInformationW = kernel32.NewProc("GetVolumeInformationW")
)

type ICONINFO struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  syscall.Handle
	HbmColor syscall.Handle
}

type BITMAP struct {
	BmType       int32
	BmWidth      int32
	BmHeight     int32
	BmWidthBytes int32
	BmPlanes     uint16
	BmBitsPixel  uint16
	BmBits       uintptr
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("\n\nPANIC RECOVERED:", r)
		}
		fmt.Println("\n\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}()

	fmt.Println("==========================================")
	fmt.Println("USB Launcher - Fyne Edition (With Icons)")
	fmt.Println("==========================================")
	fmt.Println()

	appState := &AppState{
		favorites:    make(map[string]bool),
		filterMode:   "All",
		filteredList: make([]*Executable, 0),
	}

	// Determine USB root
	exePath, err := os.Executable()
	if err != nil {
		appState.usbRoot, _ = os.Getwd()
	} else {
		appState.usbRoot = filepath.Dir(exePath)
	}
	fmt.Println("USB Root:", appState.usbRoot)

	appState.configDir = filepath.Join(appState.usbRoot, "Config")
	appState.favFile = filepath.Join(appState.configDir, "favorites.ini")
	os.MkdirAll(appState.configDir, 0755)

	appState.fsType = getFilesystemType(appState.usbRoot)
	fmt.Println("Filesystem:", appState.fsType)

	appState.loadFavorites()
	fmt.Println("Favorites loaded:", len(appState.favorites))

	// Create default icon
	appState.defaultIcon = createDefaultIcon()

	fmt.Println("Creating GUI...")
	appState.createAndRunGUI()
}

func getFilesystemType(root string) string {
	if len(root) < 2 {
		return "Unknown"
	}

	drive := root
	if root[1] == ':' {
		drive = root[:2] + "\\"
	}

	drivePtr, _ := syscall.UTF16PtrFromString(drive)
	volumeName := make([]uint16, 256)
	fsName := make([]uint16, 256)
	var serialNumber, maxComponentLen, fsFlags uint32

	ret, _, _ := procGetVolumeInformationW.Call(
		uintptr(unsafe.Pointer(drivePtr)),
		uintptr(unsafe.Pointer(&volumeName[0])), 256,
		uintptr(unsafe.Pointer(&serialNumber)),
		uintptr(unsafe.Pointer(&maxComponentLen)),
		uintptr(unsafe.Pointer(&fsFlags)),
		uintptr(unsafe.Pointer(&fsName[0])), 256,
	)

	if ret != 0 {
		return syscall.UTF16ToString(fsName)
	}
	return "Unknown"
}

func (appState *AppState) loadFavorites() {
	file, err := os.Open(appState.favFile)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inSection := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[Favorites]" {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(line, "[") {
			break
		}
		if inSection && len(line) > 0 {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) >= 1 {
				appState.favorites[parts[0]] = true
			}
		}
	}
}

func (appState *AppState) saveFavorites() {
	file, err := os.Create(appState.favFile)
	if err != nil {
		return
	}
	defer file.Close()

	file.WriteString("[Favorites]\n")
	for path := range appState.favorites {
		file.WriteString(path + "=1\n")
	}
}

func createDefaultIcon() image.Image {
	// Create a simple 16x16 default icon
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	blue := color.RGBA{70, 130, 180, 255}
	white := color.RGBA{255, 255, 255, 255}

	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if x == 0 || x == 15 || y == 0 || y == 15 {
				img.Set(x, y, white)
			} else {
				img.Set(x, y, blue)
			}
		}
	}
	return img
}

// extractIcon extracts icon from an executable file
func extractIcon(exePath string) image.Image {
	pathPtr, _ := syscall.UTF16PtrFromString(exePath)
	var hIconLarge, hIconSmall uintptr

	ret, _, _ := procExtractIconExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(unsafe.Pointer(&hIconLarge)),
		uintptr(unsafe.Pointer(&hIconSmall)),
		1,
	)

	if ret == 0 {
		return nil
	}

	// Use small icon (16x16) if available, else large
	hIcon := hIconSmall
	if hIcon == 0 {
		hIcon = hIconLarge
	}
	if hIcon == 0 {
		return nil
	}

	// Convert HICON to image.Image
	img := hIconToImage(hIcon)

	// Cleanup
	if hIconSmall != 0 {
		procDestroyIcon.Call(hIconSmall)
	}
	if hIconLarge != 0 {
		procDestroyIcon.Call(hIconLarge)
	}

	return img
}

func hIconToImage(hIcon uintptr) image.Image {
	var iconInfo ICONINFO
	ret, _, _ := procGetIconInfo.Call(hIcon, uintptr(unsafe.Pointer(&iconInfo)))
	if ret == 0 {
		return nil
	}
	defer func() {
		if iconInfo.HbmColor != 0 {
			procDeleteObject.Call(uintptr(iconInfo.HbmColor))
		}
		if iconInfo.HbmMask != 0 {
			procDeleteObject.Call(uintptr(iconInfo.HbmMask))
		}
	}()

	if iconInfo.HbmColor == 0 {
		return nil
	}

	// Get bitmap info
	var bmp BITMAP
	ret, _, _ = procGetObject.Call(
		uintptr(iconInfo.HbmColor),
		unsafe.Sizeof(bmp),
		uintptr(unsafe.Pointer(&bmp)),
	)
	if ret == 0 {
		return nil
	}

	width := int(bmp.BmWidth)
	height := int(bmp.BmHeight)
	if width <= 0 || height <= 0 || width > 256 || height > 256 {
		return nil
	}

	// Get bitmap bits
	bitsSize := width * height * 4
	bits := make([]byte, bitsSize)

	ret, _, _ = procGetBitmapBits.Call(
		uintptr(iconInfo.HbmColor),
		uintptr(bitsSize),
		uintptr(unsafe.Pointer(&bits[0])),
	)
	if ret == 0 {
		return nil
	}

	// Create image from bits (BGRA format, bottom-up)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcY := height - 1 - y
			idx := (srcY*width + x) * 4
			if idx+3 < len(bits) {
				b := bits[idx]
				g := bits[idx+1]
				r := bits[idx+2]
				a := bits[idx+3]
				if a == 0 {
					a = 255
				}
				img.SetRGBA(x, y, color.RGBA{r, g, b, a})
			}
		}
	}

	return img
}

func (appState *AppState) createAndRunGUI() {
	fmt.Println("Initializing Fyne...")

	a := app.New()
	appState.window = a.NewWindow(fmt.Sprintf("USB Discovery Launcher (%s)", appState.fsType))
	appState.window.Resize(fyne.NewSize(800, 600))

	fmt.Println("Window created")

	// Search entry
	appState.searchEntry = widget.NewEntry()
	appState.searchEntry.SetPlaceHolder("Search executables...")
	appState.searchEntry.OnChanged = func(s string) {
		appState.searchQuery = s
		appState.applyFilter()
	}

	// Filter dropdown
	appState.filterSelect = widget.NewSelect(
		[]string{"All", "★ Favorites", "Portable", "Games", "Dev", "RE"},
		func(s string) {
			appState.filterMode = s
			appState.applyFilter()
		},
	)
	appState.filterSelect.SetSelected("All")

	// Favorite button
	favButton := widget.NewButton("★ Fav", func() {
		appState.toggleFavorite()
	})

	// Launch button
	launchButton := widget.NewButton("Launch", func() {
		appState.launchSelected()
	})

	// Progress bars
	appState.scanProgress = widget.NewProgressBar()
	appState.loadProgress = widget.NewProgressBar()
	appState.statusLabel = widget.NewLabel("Initializing...")

	// Create list with icons
	appState.list = widget.NewList(
		func() int {
			appState.mutex.RLock()
			defer appState.mutex.RUnlock()
			return len(appState.filteredList)
		},
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromImage(appState.defaultIcon)
			icon.SetMinSize(fyne.NewSize(20, 20))
			icon.FillMode = canvas.ImageFillContain
			label := widget.NewLabel("Template")
			return container.NewHBox(icon, label)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			appState.mutex.RLock()
			defer appState.mutex.RUnlock()
			if int(id) < len(appState.filteredList) {
				exe := appState.filteredList[id]
				box := obj.(*fyne.Container)
				icon := box.Objects[0].(*canvas.Image)
				label := box.Objects[1].(*widget.Label)

				if exe.Icon != nil {
					icon.Image = exe.Icon
				} else {
					icon.Image = appState.defaultIcon
				}
				icon.Refresh()
				label.SetText(exe.Name)
			}
		},
	)

	// Track selection
	appState.list.OnSelected = func(id widget.ListItemID) {
		appState.mutex.Lock()
		appState.selectedID = int(id)
		appState.hasSelection = true
		appState.mutex.Unlock()
	}

	// Top bar
	topBar := container.NewBorder(
		nil, nil, nil,
		container.NewHBox(appState.filterSelect, favButton, launchButton),
		appState.searchEntry,
	)

	// Progress section
	progressSection := container.NewVBox(
		widget.NewLabel("Scanning executables..."),
		appState.scanProgress,
		widget.NewLabel("Loading launcher..."),
		appState.loadProgress,
		appState.statusLabel,
	)

	// Main layout
	content := container.NewBorder(
		container.NewVBox(topBar, widget.NewSeparator()),
		progressSection,
		nil, nil,
		appState.list,
	)

	appState.window.SetContent(content)

	// Start scanning in background
	fmt.Println("Starting scan...")
	go appState.startTwoPassScan()

	// Show and run
	fmt.Println("Running window...")
	appState.window.ShowAndRun()
	fmt.Println("Window closed")
}

func (appState *AppState) startTwoPassScan() {
	// === PASS 1: Count executables ===
	var exePaths []string
	scanQueue := []string{appState.usbRoot}
	scannedDirs := 0

	appState.statusLabel.SetText("Scanning directories...")

	for len(scanQueue) > 0 {
		current := scanQueue[0]
		scanQueue = scanQueue[1:]
		scannedDirs++

		entries, err := os.ReadDir(current)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			fullPath := filepath.Join(current, entry.Name())

			if entry.IsDir() {
				name := entry.Name()
				if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "$") ||
					strings.EqualFold(name, "System Volume Information") ||
					strings.EqualFold(name, "Config") {
					continue
				}
				scanQueue = append(scanQueue, fullPath)
			} else if strings.HasSuffix(strings.ToLower(entry.Name()), ".exe") {
				exePaths = append(exePaths, fullPath)
			}
		}

		if scannedDirs%10 == 0 {
			count := len(exePaths)
			qLen := len(scanQueue)
			pct := 0.0
			if count+qLen > 0 {
				pct = float64(count) / float64(count+qLen*2+1)
			}
			if pct > 0.99 {
				pct = 0.99
			}
			appState.scanProgress.SetValue(pct)
			appState.statusLabel.SetText(fmt.Sprintf("Found %d executables...", count))
		}
	}

	appState.totalExeCount = len(exePaths)
	appState.scanProgress.SetValue(1.0)

	if appState.totalExeCount == 0 {
		appState.statusLabel.SetText("No executables found")
		appState.loadProgress.SetValue(1.0)
		return
	}

	// === PASS 2: Load with icons ===
	appState.statusLabel.SetText(fmt.Sprintf("Loading %d executables...", appState.totalExeCount))

	batchSize := 20
	for i := 0; i < len(exePaths); i += batchSize {
		end := i + batchSize
		if end > len(exePaths) {
			end = len(exePaths)
		}

		batch := exePaths[i:end]
		var batchExes []*Executable

		for _, path := range batch {
			icon := extractIcon(path)
			if icon == nil {
				icon = appState.defaultIcon
			}
			exe := &Executable{
				Name: filepath.Base(path),
				Path: path,
				Icon: icon,
			}
			batchExes = append(batchExes, exe)
		}

		appState.mutex.Lock()
		appState.execList = append(appState.execList, batchExes...)
		appState.loadedCount = len(appState.execList)
		appState.mutex.Unlock()

		pct := float64(appState.loadedCount) / float64(appState.totalExeCount)
		appState.loadProgress.SetValue(pct)
		appState.statusLabel.SetText(fmt.Sprintf("Loaded %d / %d executables", appState.loadedCount, appState.totalExeCount))
		appState.applyFilter()

		time.Sleep(10 * time.Millisecond)
	}

	appState.statusLabel.SetText(fmt.Sprintf("Complete - %d executables loaded", appState.totalExeCount))
	appState.loadProgress.SetValue(1.0)
}

func (appState *AppState) applyFilter() {
	query := strings.ToLower(appState.searchQuery)

	appState.mutex.Lock()
	var filtered []*Executable
	for _, exe := range appState.execList {
		if query != "" && !fuzzyMatch(query, strings.ToLower(exe.Name)) {
			continue
		}
		if !appState.filterMatch(exe.Path) {
			continue
		}
		filtered = append(filtered, exe)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
	})

	appState.filteredList = filtered
	appState.mutex.Unlock()

	if appState.list != nil {
		appState.list.Refresh()
	}
}

func fuzzyMatch(needle, haystack string) bool {
	if needle == "" {
		return true
	}
	nIdx := 0
	for hIdx := 0; hIdx < len(haystack) && nIdx < len(needle); hIdx++ {
		if haystack[hIdx] == needle[nIdx] {
			nIdx++
		}
	}
	return nIdx == len(needle)
}

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

func (appState *AppState) toggleFavorite() {
	appState.mutex.RLock()
	if !appState.hasSelection {
		appState.mutex.RUnlock()
		dialog.ShowInformation("No Selection", "Please select an executable first.", appState.window)
		return
	}
	selectedID := appState.selectedID
	if selectedID >= len(appState.filteredList) {
		appState.mutex.RUnlock()
		return
	}
	exe := appState.filteredList[selectedID]
	appState.mutex.RUnlock()

	if appState.favorites[exe.Path] {
		delete(appState.favorites, exe.Path)
		dialog.ShowInformation("Favorite Removed", exe.Name+" removed from favorites", appState.window)
	} else {
		appState.favorites[exe.Path] = true
		dialog.ShowInformation("Favorite Added", exe.Name+" added to favorites", appState.window)
	}
	appState.saveFavorites()
}

func (appState *AppState) launchSelected() {
	appState.mutex.RLock()
	if !appState.hasSelection {
		appState.mutex.RUnlock()
		dialog.ShowInformation("No Selection", "Please select an executable to launch.", appState.window)
		return
	}
	selectedID := appState.selectedID
	if selectedID >= len(appState.filteredList) {
		appState.mutex.RUnlock()
		return
	}
	exe := appState.filteredList[selectedID]
	appState.mutex.RUnlock()

	fmt.Println("Launching:", exe.Path)

	cmd := exec.Command(exe.Path)
	cmd.Dir = filepath.Dir(exe.Path)
	err := cmd.Start()
	if err != nil {
		dialog.ShowError(err, appState.window)
	}
}
