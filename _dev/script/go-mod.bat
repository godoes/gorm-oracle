@echo off
color 07
title 更新并整理 GO 模块依赖
:: file-encoding=GBK
rem by iTanken

cd /d %~dp0/../../
echo 1. 更新三方依赖...
cd
:: & go get -d -u & echo.
go get -u github.com/emirpasic/gods
go get -u github.com/sijms/go-ora/v2
go get -u github.com/thoas/go-funk
go get -u gorm.io/gorm
echo.

echo 2. 整理模块依赖...
go mod tidy & echo.

:: echo 3. 导入模块依赖到 vendor 目录...
:: go mod vendor & echo.

git add .

call "%~dp0/done-time-pause.bat"
