@echo off
color 07
title 漏洞检查
:: file-encoding=GBK
rem by iTanken

rem 获取 go 版本
set gv=99999999999999999999
for /f "tokens=*" %%i in ('go version') do (
    set gv=%%i
)
set ver=%gv:~13,5%
:del-right
if "%ver:~-1%" equ "." set ver=%ver:~0,-1%&&goto del-right
if "%ver:~-1%" equ " " set ver=%ver:~0,-1%&&goto del-right
:goon
rem go 版本不能小于 1.18
if %ver% leq 1.17 (
  color 04
  echo. & echo 请使用 go1.18 或以上版本运行漏洞检查！ & echo.
  pause & exit
)

if "%~dp0" equ "%CD%\" (
  cd /d %~dp0/../../
)
echo 脚本所在路径：%~dp0
echo 当前工作目录：%CD%\
echo.
echo 开始进行漏洞检查...

echo. & echo [govulncheck]
setlocal
where govulncheck >nul 2>&1
if "%ERRORLEVEL%" equ "0" (
  echo local govulncheck...
  govulncheck ./...
) else (
  if %ver% leq 1.20 (
    echo "go1.20 latest => govulncheck@v1.1.1"
    go run golang.org/x/vuln/cmd/govulncheck@v1.1.1 ./...
  ) else (
    go run golang.org/x/vuln/cmd/govulncheck@latest ./...
  )
)
endlocal

call "%~dp0/done-time-pause.bat"
