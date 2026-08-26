@echo off
rem ---------------------------------------------------------------------------
rem bbc.bat - manage the bbc-pipeline-ui container lifecycle (Windows).
rem
rem Usage: bbc.bat <command>
rem
rem   build    Build the Docker image (Nix builds the Go binary + webapp inside)
rem   start    Build the image if missing, then run the container (webapp mode)
rem   stop     Stop and remove the container
rem   restart  Stop then start
rem   status   Show container state
rem   logs     Follow container logs
rem   help     Show this help
rem
rem Configuration (all overridable via environment variables):
rem   BBC_IMAGE        image name            (default: bbc-pipeline-ui)
rem   BBC_CONTAINER    container name        (default: bbc-pipeline-ui)
rem   BBC_CONFIG_FILE  host config path      (default: config.yml next to this script)
rem   BBC_PORT         host port to publish  (default: 8080)
rem
rem The host config file is mounted at /config.yml inside the container.
rem ---------------------------------------------------------------------------

setlocal enabledelayedexpansion

set "SCRIPT_DIR=%~dp0"

if not defined BBC_IMAGE (
  set "IMAGE_NAME=bbc-pipeline-ui"
) else (
  set "IMAGE_NAME=%BBC_IMAGE%"
)

if not defined BBC_CONTAINER (
  set "CONTAINER_NAME=bbc-pipeline-ui"
) else (
  set "CONTAINER_NAME=%BBC_CONTAINER%"
)

if not defined BBC_CONFIG_FILE (
  set "CONFIG_FILE=%SCRIPT_DIR%config.yml"
) else (
  set "CONFIG_FILE=%BBC_CONFIG_FILE%"
)

if not defined BBC_PORT (
  set "PORT=8080"
) else (
  set "PORT=%BBC_PORT%"
)

if "%~1"==""       goto usage
if /i "%~1"=="build"   goto build
if /i "%~1"=="start"   goto start
if /i "%~1"=="stop"    goto stop
if /i "%~1"=="restart" goto restart
if /i "%~1"=="status"  goto status
if /i "%~1"=="logs"    goto logs
if /i "%~1"=="help"    goto usage

echo [bbc] Unknown command: %~1
goto usage

rem ---------------------------------------------------------------------------
:build
echo [bbc] Building image '%IMAGE_NAME%' ...
docker build -t "%IMAGE_NAME%" "%SCRIPT_DIR%"
if errorlevel 1 exit /b 1
echo [bbc] Image '%IMAGE_NAME%' built.
goto :eof

rem ---------------------------------------------------------------------------
:ensure_config
if exist "%CONFIG_FILE%" goto :eof

:ensure_config_prompt
echo.
echo [bbc] Config file not found: %CONFIG_FILE%
echo.
echo   1. Run the first-time setup wizard
echo   2. Set BBC_CONFIG_FILE to another path
echo   q. Abort
echo.
set "CHOICE="
set /p "CHOICE=Choose an option [1/2/q]: "

if "%CHOICE%"=="1" (
  call :run_wizard
  goto :ensure_config
)
if "%CHOICE%"=="2" (
  call :set_config_path
  goto :ensure_config
)
if /i "%CHOICE%"=="q" goto :abort
if "%CHOICE%"=="" goto :abort

echo [bbc] Unknown option.
goto :ensure_config_prompt

:abort
echo [bbc] Aborted.
exit /b 1

:set_config_path
set "NEWPATH="
set /p "NEWPATH=Config file path: "
if "%NEWPATH%"=="" (
  echo [bbc] No path given.
  goto :eof
)
set "CONFIG_FILE=%NEWPATH%"
goto :eof

:run_wizard
docker image inspect "%IMAGE_NAME%" >nul 2>nul
if errorlevel 1 (
  echo [bbc] Image not found - building it now.
  call :build
  if errorlevel 1 exit /b 1
)

set "CFG_DIR=%~dp0"
set "CFG_BASE=config.yml"
for %%F in ("%CONFIG_FILE%") do (
  if not "%%~dpF"=="" set "CFG_DIR=%%~dpF"
  set "CFG_BASE=%%~nxF"
)
if "%CFG_DIR:~-1%"=="\" set "CFG_DIR=%CFG_DIR:~0,-1%"

echo [bbc] Starting the interactive setup wizard...
echo [bbc] Config will be written to %CONFIG_FILE%.
echo [bbc] After the wizard completes, quit the TUI with 'q'.

docker run --rm -it -v "%CFG_DIR%:/cfg" -e "BBC_CONFIG=/cfg/%CFG_BASE%" "%IMAGE_NAME%"

if exist "%CONFIG_FILE%" (
  echo [bbc] Configuration saved to %CONFIG_FILE%.
) else (
  echo [bbc] Wizard finished but %CONFIG_FILE% was not created.
)
goto :eof

rem ---------------------------------------------------------------------------
:start
call :ensure_config
if errorlevel 1 exit /b 1

docker image inspect "%IMAGE_NAME%" >nul 2>nul
if errorlevel 1 (
  echo [bbc] Image not found - building it now.
  call :build
  if errorlevel 1 exit /b 1
)

set "STATE="
for /f "tokens=*" %%s in ('docker inspect --format "{{.State.Status}}" "%CONTAINER_NAME%" 2^>nul') do set "STATE=%%s"

if "%STATE%"=="running" (
  echo [bbc] Container '%CONTAINER_NAME%' is already running.
  goto :eof
)

docker container inspect "%CONTAINER_NAME%" >nul 2>nul
if not errorlevel 1 (
  echo [bbc] Removing stopped container '%CONTAINER_NAME%' ...
  docker rm -f "%CONTAINER_NAME%" >nul
)

echo [bbc] Starting container '%CONTAINER_NAME%' (webapp on http://localhost:%PORT%) ...
docker run -d ^
  --name "%CONTAINER_NAME%" ^
  --restart unless-stopped ^
  -p "%PORT%:8080" ^
  -v "%CONFIG_FILE%:/config.yml" ^
  "%IMAGE_NAME%" ^
  --webapp "0.0.0.0:8080"

if errorlevel 1 exit /b 1
echo [bbc] Started. Webapp: http://localhost:%PORT%  (config: %CONFIG_FILE%)
goto :eof

rem ---------------------------------------------------------------------------
:stop
docker container inspect "%CONTAINER_NAME%" >nul 2>nul
if errorlevel 1 (
  echo [bbc] Container '%CONTAINER_NAME%' does not exist.
  goto :eof
)
echo [bbc] Stopping container '%CONTAINER_NAME%' ...
docker stop "%CONTAINER_NAME%" >nul
docker rm "%CONTAINER_NAME%" >nul
echo [bbc] Stopped and removed.
goto :eof

rem ---------------------------------------------------------------------------
:restart
call :stop
call :start
goto :eof

rem ---------------------------------------------------------------------------
:status
docker container inspect "%CONTAINER_NAME%" >nul 2>nul
if errorlevel 1 (
  echo [bbc] Container '%CONTAINER_NAME%' does not exist ^(not created^).
  goto :eof
)
echo [bbc] Container: %CONTAINER_NAME%
docker ps -a --filter "name=^/%CONTAINER_NAME%$"
goto :eof

rem ---------------------------------------------------------------------------
:logs
docker container inspect "%CONTAINER_NAME%" >nul 2>nul
if errorlevel 1 (
  echo [bbc] Container '%CONTAINER_NAME%' does not exist.
  exit /b 1
)
docker logs -f "%CONTAINER_NAME%"
goto :eof

rem ---------------------------------------------------------------------------
:usage
echo.
echo bbc.bat - manage the bbc-pipeline-ui container lifecycle
echo.
echo Usage: bbc.bat ^<command^>
echo.
echo   build    Build the Docker image
echo   start    Build if missing, then run the container (webapp mode)
echo   stop     Stop and remove the container
echo   restart  Stop then start
echo   status   Show container state
echo   logs     Follow container logs
echo   help     Show this help
echo.
echo Config: BBC_IMAGE, BBC_CONTAINER, BBC_CONFIG_FILE, BBC_PORT
echo (default config: %CONFIG_FILE%)
goto :eof
