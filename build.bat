@echo off
REM ============================================
REM USB Launcher (Fyne) - Build Script
REM ============================================
REM IMPORTANT: Fyne requires CGO and a C compiler!
REM 
REM You need to install one of these:
REM   - TDM-GCC: https://jmeubank.github.io/tdm-gcc/
REM   - MinGW-w64: https://www.mingw-w64.org/
REM   - MSYS2: https://www.msys2.org/
REM
REM After installing, ensure gcc is in your PATH.
REM ============================================

echo.
echo ========================================
echo USB Launcher (Fyne) Build Script
echo ========================================
echo.

REM Check for Go
where go >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Go is not in PATH
    pause
    exit /b 1
)

echo [1/4] Go found:
go version
echo.

REM Check for GCC (required for CGO)
where gcc >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: GCC not found!
    echo.
    echo Fyne requires a C compiler. Please install one of:
    echo   - TDM-GCC: https://jmeubank.github.io/tdm-gcc/
    echo   - MinGW-w64: https://www.mingw-w64.org/
    echo.
    echo After installing, add it to your PATH and try again.
    echo.
    pause
    exit /b 1
)

echo [2/4] GCC found:
gcc --version | findstr /C:"gcc"
echo.

REM Enable CGO
set CGO_ENABLED=1

REM Download dependencies (this takes a while first time)
echo [3/5] Downloading dependencies...
echo       (Fyne is large, this may take several minutes)
go mod tidy
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Failed to download dependencies
    pause
    exit /b 1
)

REM Compile Windows resource file (embeds icon into .exe)
echo [4/5] Compiling resource file (icon)...
windres -o resource.syso resource.rc
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: windres failed - ensure MinGW/TDM-GCC is installed
    pause
    exit /b 1
)

REM Build
echo [5/5] Building executables...

echo Building DEBUG version (with console)...
go build -o USBLauncher_Fyne_Debug.exe .
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Debug build failed
    pause
    exit /b 1
)

echo Building RELEASE version (no console)...
go build -ldflags="-H windowsgui -s -w" -o USBLauncher_Fyne.exe .
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Release build failed
    pause
    exit /b 1
)

echo.
echo ========================================
echo BUILD SUCCESSFUL!
echo ========================================
echo.
echo Files created:
echo   USBLauncher_Fyne_Debug.exe - Debug (with console)
echo   USBLauncher_Fyne.exe       - Release (no console)
echo.
for %%A in (USBLauncher_Fyne.exe) do echo Size: %%~zA bytes
echo.
pause
