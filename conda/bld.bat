if not exist "%PREFIX%\bin" ( 
    mkdir "%PREFIX%\bin"
    if errorlevel 1 exit 1
)

copy %SRC_DIR%\gocmd %PREFIX%\bin\gocmd
