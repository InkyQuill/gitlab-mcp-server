@echo off
REM Setup script for Windows
REM Installs prerequisites and optionally runs the MCP installer

setlocal enabledelayedexpansion

set GO_VERSION_MIN=1.23
set BINARY_NAME=gitlab-mcp-server
set INSTALLER_SCRIPT=scripts\install.js

echo === GitLab MCP Server Setup ===
echo.

REM Check for Go
echo Checking for Go...
where go >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo Error: Go is not installed.
    echo Please install Go %GO_VERSION_MIN% or later from https://go.dev/dl/
    exit /b 1
)

for /f "tokens=3" %%i in ('go version') do set GO_VERSION=%%i
echo Go found:
go version

REM Download dependencies
echo.
echo Downloading dependencies...
go mod download
if %ERRORLEVEL% NEQ 0 (
    echo Error: Failed to download dependencies
    exit /b 1
)

echo.
echo Prerequisites installed successfully!
echo.

REM Ask if user wants to run installer
set /p RUN_INSTALLER="Do you want to configure MCP servers now? (y/n) "
if /i "%RUN_INSTALLER%"=="y" (
    REM Check for Node.js
    echo Checking for Node.js...
    where node >nul 2>&1
    if %ERRORLEVEL% NEQ 0 (
        echo Error: Node.js is not installed.
        echo Please install Node.js to run the MCP installer.
        exit /b 1
    )
    echo Node.js found:
    node --version

    REM Build the main binary first
    echo.
    echo Building GitLab MCP server binary...
    if not exist bin mkdir bin
    go build -o "bin\%BINARY_NAME%.exe" ./cmd/gitlab-mcp-server

    if exist "bin\%BINARY_NAME%.exe" (
        echo Binary built successfully!
        echo.
        echo Running MCP installer...
        node "%INSTALLER_SCRIPT%"
    ) else (
        echo Error: Failed to build binary
        exit /b 1
    )
) else (
    echo You can run the installer later with: make install-mcp
)

endlocal

