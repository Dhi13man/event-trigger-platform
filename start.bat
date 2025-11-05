@echo off
REM Event Trigger Platform - Startup Script (Windows)
REM Starts all services via Docker Compose

echo Starting Event Trigger Platform...
echo.
echo This will start:
echo   - MySQL (localhost:3306)
echo   - Kafka + Zookeeper
echo   - API Server (localhost:8080)
echo   - Scheduler
echo   - Workers (2 replicas)
echo   - Retention Manager
echo.

REM Check if Docker is running
docker info >nul 2>&1
if errorlevel 1 (
    echo Error: Docker is not running
    echo Please start Docker Desktop and try again
    exit /b 1
)

REM Build and start services
echo Building Docker images...
docker compose -f deploy\docker-compose.yml build

echo.
echo Starting services...
docker compose -f deploy\docker-compose.yml up -d

echo.
echo Waiting for services to be healthy...
timeout /t 5 /nobreak >nul

REM Show service status
docker compose -f deploy\docker-compose.yml ps

echo.
echo Event Trigger Platform is running!
echo.
echo Access points:
echo    API:     http://localhost:8080
echo    Health:  http://localhost:8080/health
echo    Metrics: http://localhost:8080/metrics
echo.
echo Useful commands:
echo    View logs:  docker compose -f deploy\docker-compose.yml logs -f
echo    Stop all:   docker compose -f deploy\docker-compose.yml down
echo    Restart:    docker compose -f deploy\docker-compose.yml restart
echo.
pause
