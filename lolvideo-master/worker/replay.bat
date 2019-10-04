@echo off

set length=%1
set mode=%2
set server=%3
set key=%4
set game=%5
set platform=%6
set startupDelay=%7
set champ=%8
set noRecord=%9
set outputPath=%10
shift
shift
shift
shift
shift
shift
shift
shift
shift
shift

setlocal enabledelayedexpansion
set RADS_PATH="C:\Riot Games\League of Legends\RADS"
@cd /d "%RADS_PATH%\solutions\lol_game_client_sln\releases"

set init=0
set v0=0&set v1=0&set v2=0&set v3=0
for /f "delims=" %%F in ('dir /a:d /b') do (
	for /F "tokens=1,2,3,4 delims=." %%i in ("%%F") do (
		if !init! equ 0 ( set init=1&set flag=1 ) else (
			set flag=0
			
			if %%i gtr !v0! ( set flag=1 ) else (
				if %%j gtr !v1! ( set flag=1 ) else (
					if %%k gtr !v2! ( set flag=1 ) else (
						if %%l gtr !v3! ( set flag=1 )
					)
				)
			)
		)
		
		if !flag! gtr 0 (
			set v0=%%i&set v1=%%j&set v2=%%k&set v3=%%l
		)
	)
)

if !init! equ 0 goto cannotFind
set lolver=!v0!.!v1!.!v2!.!v3!

@cd /d "!RADS_PATH!\solutions\lol_game_client_sln\releases\!lolver!\deploy"
if exist "League of Legends.exe" (
	del c:\needs_bootstrap.txt
	@start "" "League of Legends.exe" "8394" "LoLLauncher.exe" "" "%mode% %server% %key% %game% %platform%" -UseRads
	
	@timeout %startupDelay%

	if %noRecord% equ 0 (
	   @start /b C:\Users\Administrator\Desktop\ffmpeg -y -f gdigrab -framerate 25 -i desktop -t %length% -b:v 1000k z:\%game%-%platform%-%champ%.mp4
	)

	@timeout 10
	@timeout %length%
	@taskkill /IM "League of Legends.exe" /F
	@taskkill /IM "EloBuddy.Loader.exe" /F /S localhost

	goto exit
)
:cannotFind
echo Cannot find LOL directory path
goto exit
:exit

