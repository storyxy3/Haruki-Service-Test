@echo off
set PYTHON_EXE=D:\tool\Python_Ontology\python.exe

echo Starting Haruki-Drawing-API...
cd /d D:\github\Haruki-Drawing-API
echo Using Python at: %PYTHON_EXE%
start "Drawing API" "%PYTHON_EXE%" src/core/main.py

echo Waiting for API to start (10 seconds)...
timeout /t 10 /nobreak

echo Running Go Test Client...
cd /d D:\github\work\lunabot-service-go
go run cmd/test_cli/main.go -cmd "190"

echo.
echo If successful, check D:\github\testfile for the generated image.
pause
