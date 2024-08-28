@echo off
color 07
title 静态检查
:: file-encoding=GBK
rem by iTanken

echo.
if "%~dp0" equ "%CD%\" (
  cd /d %~dp0/../../
)
echo 脚本所在路径：%~dp0
echo 当前工作目录：%CD%\
echo.
echo 开始进行静态检查...

echo. & echo [golangci-lint.run]
setlocal
where golangci-lint >nul 2>&1
if "%ERRORLEVEL%" equ "0" (
  echo local golangci-lint...
  golangci-lint run --timeout=5m
) else (
  echo "go1.20 latest => golangci-lint@v1.55.2"
  go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2 run --timeout=5m
)
endlocal

rem echo. & echo [staticcheck.io]
rem go run honnef.co/go/tools/cmd/staticcheck@latest -f text ./...

call "%~dp0/done-time-pause.bat"
