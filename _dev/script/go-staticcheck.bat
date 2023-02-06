@echo off
color 07
title 静态检查
:: file-encoding=GBK
rem by iTanken
echo 开始进行静态检查... & echo.

cd /d %~dp0/../../
echo [revive.run]
go run github.com/mgechev/revive@latest -config ./_dev/config/revive.toml -exclude ./vendor/... -formatter stylish ./...
echo. & echo [staticcheck.io]
go run honnef.co/go/tools/cmd/staticcheck@latest -f text ./...

call "%~dp0/done-time-pause.bat"
