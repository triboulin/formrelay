@echo off
setlocal enabledelayedexpansion

if /I "%~1"=="-h" goto :help
if /I "%~1"=="--help" goto :help

set "VERSION_FILE=VERSION"
set "IMAGE_REPO=registry.triboulin.fr/formrelay-admin"

if not exist "%VERSION_FILE%" (
  echo Missing %VERSION_FILE% file.
  exit /b 1
)

set /p CURRENT_VERSION=<"%VERSION_FILE%"
for /f "tokens=* delims= " %%A in ("%CURRENT_VERSION%") do set "CURRENT_VERSION=%%A"

for /f "tokens=1-3 delims=." %%A in ("%CURRENT_VERSION%") do (
  set "MAJOR=%%A"
  set "MINOR=%%B"
  set "PATCH=%%C"
)

if "%MAJOR%"=="" goto :invalid_version
if "%MINOR%"=="" goto :invalid_version
if "%PATCH%"=="" goto :invalid_version

set /a MAJOR_NUM=MAJOR+0 >nul 2>&1
if errorlevel 1 goto :invalid_version
set /a MINOR_NUM=MINOR+0 >nul 2>&1
if errorlevel 1 goto :invalid_version
set /a PATCH_NUM=PATCH+0 >nul 2>&1
if errorlevel 1 goto :invalid_version

set /a PATCH_NUM=PATCH_NUM+1
set "NEW_VERSION=%MAJOR_NUM%.%MINOR_NUM%.%PATCH_NUM%"

>"%VERSION_FILE%" echo %NEW_VERSION%

set "IMAGE=%IMAGE_REPO%:%NEW_VERSION%"

echo [1/2] Building image %IMAGE%...
docker build -t "%IMAGE%" .
if errorlevel 1 (
  echo Build failed.
  exit /b 1
)

echo [2/2] Pushing image %IMAGE%...
docker push "%IMAGE%"
if errorlevel 1 (
  echo Push failed.
  exit /b 1
)

echo Done: %IMAGE%
echo VERSION updated to %NEW_VERSION%
exit /b 0

:invalid_version
echo Invalid VERSION format. Expected MAJOR.MINOR.PATCH (example: 1.0.0).
exit /b 1

:help
echo Usage: %~nx0
echo.
echo Reads VERSION, auto-increments PATCH, then builds and pushes:
echo   registry.triboulin.fr/formrelay-admin:^<new-version^>
echo.
echo Example:
echo   %~nx0
exit /b 0
