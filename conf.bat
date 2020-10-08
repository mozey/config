set APP_DIR=%cd%

%GOPATH%/bin/configu > conf.tmp.bat
conf.tmp.bat
rm conf.tmp.bat
