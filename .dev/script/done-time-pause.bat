@echo off
:: file-encoding=GBK
rem by iTanken
rem 公共 footer 脚本，用于打印完成时间及根据 BatNoPause 环境变量判断是否显示按任意键继续

color 0f
rem 获取当前时间
set _now=%date:~0,4%-%date:~5,2%-%date:~8,2% %time:~0,2%:%time:~3,2%:%time:~6,2%.%time:~9,3%

rem 获取窗口标题
for /f "usebackq delims=" %%t in (`powershell -noprofile -c "[Console]::Title.Replace(' - '+[Environment]::CommandLine,'')"`) do (
  set _title=%%t
)

rem 打印完成信息
echo. & echo [%_now%] 执行%_title%完成！ & echo.
if "%BatNoPause%" NEQ "1" (
  pause
)
