// USB Discovery Launcher - Go Edition (Imperative Walk API)
// Features: Two-pass scan, icon extraction, fuzzy search, favorites, view toggle

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// Executable represents a found .exe file
type Executable struct {
	Name      string
	Path      string
	IconIndex int32
}

// ExeModel implements walk.ListModel for the ListBox
type ExeModel struct {
	walk.ListModelBase
	items []*Executable
}

func (m *ExeModel) ItemCount() int {
	return len(m.items)
}

func (m *ExeModel) Value(index int) interface{} {
	if index < 0 || index >= len(m.items) {
		return ""
	}
	return m.items[index].Name
}

// AppState holds all application state
type AppState struct {
	execList     []*Executable
	favorites    map[string]bool
	filterMode   string
	usbRoot      string
	configDir    string
	favFile      string
	fsType       string

	// Scan state
	totalExeCount int
	loadedCount   int

	// GUI elements
	mainWindow   *walk.MainWindow
	listBox      *walk.ListBox
	searchBox    *walk.LineEdit
	filterCombo  *walk.ComboBox
	scanProgress *walk.ProgressBar
	loadProgress *walk.ProgressBar
	scanLabel    *walk.Label
	loadLabel    *walk.Label
	statusLabel  *walk.Label

	// Model
	model        *ExeModel
	filteredList []*Executable

	// Synchronization
	mutex sync.RWMutex
}

// Windows API for icon extraction
var (
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procExtractIconEx = shell32.NewProc("ExtractIconExW")
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("\n\nPANIC RECOVERED:", r)
		}
		fmt.Println("\n\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}()

	fmt.Println("==========================================")
	fmt.Println("USB Launcher Debug Mode")
	fmt.Println("==========================================")
	fmt.Println()

	app := &AppState{
		favorites:    make(map[string]bool),
		filterMode:   "All",
		filteredList: make([]*Executable, 0),
	}

	// Determine USB root (script directory)
	exePath, err := os.Executable()
	if err != nil {
		app.usbRoot, _ = os.Getwd()
		fmt.Println("Using working directory:", app.usbRoot)
	} else {
		app.usbRoot = filepath.Dir(exePath)
		fmt.Println("Using executable directory:", app.usbRoot)
	}

	app.configDir = filepath.Join(app.usbRoot, "Config")
	app.favFile = filepath.Join(app.configDir, "favorites.ini")

	// Create config directory
	os.MkdirAll(app.configDir, 0755)

	// Detect filesystem type
	app.fsType = getFilesystemType(app.usbRoot)
	fmt.Println("Filesystem type:", app.fsType)

	// Load favorites
	app.loadFavorites()
	fmt.Println("Favorites loaded:", len(app.favorites))

	// Create and run GUI
	fmt.Println("Creating GUI...")
	app.createAndRunGUI()
}

func getFilesystemType(root string) string {
	if len(root) < 1 {
		return "Unknown"
	}

	drive := root
	if len(root) >= 2 && root[1] == ':' {
		drive = root[:2] + "\\"
	} else {
		drive = root[:1] + ":\\"
	}

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getVolumeInfo := kernel32.NewProc("GetVolumeInformationW")

	volumeName := make([]uint16, 256)
	fsName := make([]uint16, 256)
	var serialNumber, maxComponentLen, fsFlags uint32

	drivePtr, _ := syscall.UTF16PtrFromString(drive)

	ret, _, _ := getVolumeInfo.Call(
		uintptr(unsafe.Pointer(drivePtr)),
		uintptr(unsafe.Pointer(&volumeName[0])),
		256,
		uintptr(unsafe.Pointer(&serialNumber)),
		uintptr(unsafe.Pointer(&maxComponentLen)),
		uintptr(unsafe.Pointer(&fsFlags)),
		uintptr(unsafe.Pointer(&fsName[0])),
		256,
	)

	if ret != 0 {
		return syscall.UTF16ToString(fsName)
	}
	return "Unknown"
}

func (app *AppState) loadFavorites() {
	file, err := os.Open(app.favFile)
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
				app.favorites[parts[0]] = true
			}
		}
	}
}

func (app *AppState) saveFavorites() {
	file, err := os.Create(app.favFile)
	if err != nil {
		return
	}
	defer file.Close()

	file.WriteString("[Favorites]\n")
	for path := range app.favorites {
		file.WriteString(path + "=1\n")
	}
}

func (app *AppState) createAndRunGUI() {
	var err error

	fmt.Println("Creating main window...")
	app.mainWindow, err = walk.NewMainWindow()
	if err != nil {
		fmt.Println("ERROR creating MainWindow:", err)
		return
	}

	title := fmt.Sprintf("USB Discovery Launcher (%s)", app.fsType)
	app.mainWindow.SetTitle(title)
	app.mainWindow.SetSize(walk.Size{Width: 750, Height: 550})

	// Set up layout
	fmt.Println("Setting up layout...")
	vbox := walk.NewVBoxLayout()
	vbox.SetMargins(walk.Margins{HNear: 10, VNear: 10, HFar: 10, VFar: 10})
	vbox.SetSpacing(5)
	app.mainWindow.SetLayout(vbox)

	// Warning label for FAT32/exFAT
	if app.fsType == "FAT32" || app.fsType == "exFAT" {
		warnLabel, _ := walk.NewLabel(app.mainWindow)
		if app.fsType == "FAT32" {
			warnLabel.SetText("Warning: FAT32 detected (4GB file limit)")
		} else {
			warnLabel.SetText("Notice: exFAT detected (symlinks may fail)")
		}
		warnLabel.SetTextColor(walk.RGB(255, 0, 0))
	}

	// === Top toolbar composite ===
	fmt.Println("Creating toolbar...")
	toolbarComposite, _ := walk.NewComposite(app.mainWindow)
	hbox := walk.NewHBoxLayout()
	hbox.SetMargins(walk.Margins{})
	hbox.SetSpacing(5)
	toolbarComposite.SetLayout(hbox)

	// Search box
	app.searchBox, _ = walk.NewLineEdit(toolbarComposite)
	app.searchBox.SetCueBanner("Search executables...")
	app.searchBox.TextChanged().Attach(func() {
		app.applyFilter()
	})

	// Filter combo box
	app.filterCombo, _ = walk.NewComboBox(toolbarComposite)
	app.filterCombo.SetModel([]string{"All", "★ Favorites", "Portable", "Games", "Dev", "RE"})
	app.filterCombo.SetCurrentIndex(0)
	app.filterCombo.CurrentIndexChanged().Attach(func() {
		app.filterMode = app.filterCombo.Text()
		app.applyFilter()
	})

	// Favorite button
	favButton, _ := walk.NewPushButton(toolbarComposite)
	favButton.SetText("★ Fav")
	favButton.Clicked().Attach(func() {
		app.toggleFavorite()
	})

	// Launch button
	launchButton, _ := walk.NewPushButton(toolbarComposite)
	launchButton.SetText("Launch")
	launchButton.Clicked().Attach(func() {
		app.launchSelected()
	})

	// === List Box ===
	fmt.Println("Creating list box...")
	app.listBox, _ = walk.NewListBox(app.mainWindow)
	app.listBox.SetMinMaxSize(walk.Size{Width: 0, Height: 200}, walk.Size{})
	
	// Double-click to launch
	app.listBox.ItemActivated().Attach(func() {
		app.launchSelected()
	})

	// Initialize model
	app.model = &ExeModel{items: []*Executable{}}
	app.listBox.SetModel(app.model)

	// === Progress section composite ===
	fmt.Println("Creating progress section...")
	progressComposite, _ := walk.NewComposite(app.mainWindow)
	pbox := walk.NewVBoxLayout()
	pbox.SetMargins(walk.Margins{})
	pbox.SetSpacing(2)
	progressComposite.SetLayout(pbox)

	// Scan progress
	app.scanLabel, _ = walk.NewLabel(progressComposite)
	app.scanLabel.SetText("Scanning executables... (0%)")

	app.scanProgress, _ = walk.NewProgressBar(progressComposite)
	app.scanProgress.SetRange(0, 100)
	app.scanProgress.SetValue(0)

	// Load progress
	app.loadLabel, _ = walk.NewLabel(progressComposite)
	app.loadLabel.SetText("Loading launcher... (0%)")

	app.loadProgress, _ = walk.NewProgressBar(progressComposite)
	app.loadProgress.SetRange(0, 100)
	app.loadProgress.SetValue(0)

	// Status label
	app.statusLabel, _ = walk.NewLabel(progressComposite)
	app.statusLabel.SetText("Initializing...")

	// Start scanning in background
	fmt.Println("Starting background scan...")
	go app.startTwoPassScan()

	// Show window and run
	fmt.Println("Showing window...")
	app.mainWindow.SetVisible(true)

	fmt.Println("Running message loop...")
	app.mainWindow.Run()

	fmt.Println("Window closed")
}

func (app *AppState) startTwoPassScan() {
	// === PASS 1: Count all executables ===
	var exePaths []string
	scanQueue := []string{app.usbRoot}
	scannedDirs := 0

	app.updateStatus("Scanning directories...")

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
				// Skip hidden, system, and special directories
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

		// Update scan progress periodically
		if scannedDirs%10 == 0 {
			count := len(exePaths)
			qLen := len(scanQueue)
			pct := 0
			if qLen > 0 {
				pct = (count * 100) / (count + qLen*2)
			} else {
				pct = 99
			}
			if pct > 99 {
				pct = 99
			}
			app.updateScanProgress(pct, count)
		}
	}

	app.totalExeCount = len(exePaths)
	app.updateScanProgress(100, app.totalExeCount)

	if app.totalExeCount == 0 {
		app.updateStatus("No executables found")
		app.updateLoadProgress(100, 0, 0)
		return
	}

	// === PASS 2: Load executables ===
	app.updateStatus(fmt.Sprintf("Loading %d executables...", app.totalExeCount))

	batchSize := 50
	for i := 0; i < len(exePaths); i += batchSize {
		end := i + batchSize
		if end > len(exePaths) {
			end = len(exePaths)
		}

		batch := exePaths[i:end]
		var batchExes []*Executable

		for _, path := range batch {
			exe := &Executable{
				Name: filepath.Base(path),
				Path: path,
			}
			batchExes = append(batchExes, exe)
		}

		// Add batch to list
		app.mutex.Lock()
		app.execList = append(app.execList, batchExes...)
		app.loadedCount = len(app.execList)
		app.mutex.Unlock()

		// Update UI
		pct := (app.loadedCount * 100) / app.totalExeCount
		app.updateLoadProgress(pct, app.loadedCount, app.totalExeCount)

		// Refresh the list
		app.applyFilter()

		// Small delay to keep UI responsive
		time.Sleep(5 * time.Millisecond)
	}

	app.updateStatus(fmt.Sprintf("Complete - %d executables loaded", app.totalExeCount))
	app.updateLoadProgress(100, app.totalExeCount, app.totalExeCount)
}

func (app *AppState) updateScanProgress(pct int, count int) {
	app.mainWindow.Synchronize(func() {
		app.scanProgress.SetValue(pct)
		app.scanLabel.SetText(fmt.Sprintf("Scanning executables... (%d%%)", pct))
		app.statusLabel.SetText(fmt.Sprintf("Found %d executables...", count))
	})
}

func (app *AppState) updateLoadProgress(pct int, loaded int, total int) {
	app.mainWindow.Synchronize(func() {
		app.loadProgress.SetValue(pct)
		app.loadLabel.SetText(fmt.Sprintf("Loading launcher... (%d%%)", pct))
		if total > 0 {
			app.statusLabel.SetText(fmt.Sprintf("Loaded %d / %d executables", loaded, total))
		}
	})
}

func (app *AppState) updateStatus(msg string) {
	app.mainWindow.Synchronize(func() {
		app.statusLabel.SetText(msg)
	})
}

func (app *AppState) applyFilter() {
	query := strings.ToLower(app.searchBox.Text())

	app.mutex.Lock()
	var filtered []*Executable
	for _, exe := range app.execList {
		// Apply search filter
		if query != "" && !fuzzyMatch(query, strings.ToLower(exe.Name)) {
			continue
		}

		// Apply category filter
		if !app.filterMatch(exe.Path) {
			continue
		}

		filtered = append(filtered, exe)
	}

	// Sort by name
	sort.Slice(filtered, func(i, j int) bool {
		return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
	})

	app.filteredList = filtered
	app.model.items = filtered
	app.mutex.Unlock()

	// Update list on UI thread
	app.mainWindow.Synchronize(func() {
		app.model.PublishItemsReset()
	})
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

func (app *AppState) toggleFavorite() {
	idx := app.listBox.CurrentIndex()
	if idx < 0 {
		return
	}

	app.mutex.RLock()
	if idx >= len(app.filteredList) {
		app.mutex.RUnlock()
		return
	}
	exe := app.filteredList[idx]
	app.mutex.RUnlock()

	if app.favorites[exe.Path] {
		delete(app.favorites, exe.Path)
		walk.MsgBox(app.mainWindow, "Favorite Removed", exe.Name+" removed from favorites", walk.MsgBoxIconInformation)
	} else {
		app.favorites[exe.Path] = true
		walk.MsgBox(app.mainWindow, "Favorite Added", exe.Name+" added to favorites", walk.MsgBoxIconInformation)
	}
	app.saveFavorites()
}

func (app *AppState) launchSelected() {
	idx := app.listBox.CurrentIndex()
	if idx < 0 {
		walk.MsgBox(app.mainWindow, "No Selection", "Please select an executable to launch.", walk.MsgBoxIconWarning)
		return
	}

	app.mutex.RLock()
	if idx >= len(app.filteredList) {
		app.mutex.RUnlock()
		return
	}
	exe := app.filteredList[idx]
	app.mutex.RUnlock()

	fmt.Println("Launching:", exe.Path)

	// Use ShellExecute to launch
	verb, _ := syscall.UTF16PtrFromString("open")
	file, _ := syscall.UTF16PtrFromString(exe.Path)
	dir, _ := syscall.UTF16PtrFromString(filepath.Dir(exe.Path))

	shellExecute := shell32.NewProc("ShellExecuteW")
	ret, _, _ := shellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		0,
		uintptr(unsafe.Pointer(dir)),
		uintptr(win.SW_SHOWNORMAL),
	)

	if ret <= 32 {
		walk.MsgBox(app.mainWindow, "Error", "Failed to launch "+exe.Name, walk.MsgBoxIconError)
	}
}
