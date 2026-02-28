@echo off
REM ============================================
REM USB Launcher v2 - Build Script
REM ============================================

echo.
echo ========================================
echo USB Launcher Build Script
echo ========================================
echo.

where go >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Go is not in PATH
    pause
    exit /b 1
)

echo [1/3] Go found: 
go version
echo.

echo [2/3] Downloading dependencies...
go mod tidy
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Failed to download dependencies
    pause
    exit /b 1
)

echo [3/3] Building executables...

echo Building DEBUG version...
go build -o USBLauncher_Debug.exe .
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Debug build failed
    pause
    exit /b 1
)

echo Building RELEASE version...
go build -ldflags="-H windowsgui -s -w" -o USBLauncher.exe .
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
echo   USBLauncher_Debug.exe          - Debug version (shows console)
echo   USBLauncher_Debug.exe.manifest - Required manifest
echo   USBLauncher.exe                - Release version (no console)
echo   USBLauncher.exe.manifest       - Required manifest
echo.
echo IMPORTANT: Keep .exe and .manifest files together!
echo.
for %%A in (USBLauncher.exe) do echo Size: %%~zA bytes
echo.
pause
