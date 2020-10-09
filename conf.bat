set APP_DIR=%cd%

%GOPATH%\bin\configu > %APP_DIR%\conf.tmp.bat
call %APP_DIR%\conf.tmp.bat
del /f %APP_DIR%\conf.tmp.bat

REM done
