set APP_DIR=%cd%
set ENV=%1
IF "%ENV%"=="" set ENV=dev

%GOPATH%\bin\configu -os windows -env %ENV% > %APP_DIR%\conf.tmp.bat
if %errorlevel% neq 0 exit /b %errorlevel%
call %APP_DIR%\conf.tmp.bat
del /f %APP_DIR%\conf.tmp.bat

REM done
