# Пример сборки с передачей информации о версии через ldflags для Windows

$VERSION = "v1.0.0"
$DATE = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$COMMIT = try { git rev-parse --short HEAD } catch { "unknown" }

Write-Host "Building shortener with build info:"
Write-Host "Version: $VERSION"
Write-Host "Date: $DATE" 
Write-Host "Commit: $COMMIT"

# Сборка shortener
go build -ldflags "-X main.buildVersion=$VERSION -X main.buildDate=$DATE -X main.buildCommit=$COMMIT" -o shortener.exe ./cmd/shortener

# Сборка api  
go build -ldflags "-X main.buildVersion=$VERSION -X main.buildDate=$DATE -X main.buildCommit=$COMMIT" -o api.exe ./cmd/api

Write-Host "Build completed!"
Write-Host ""
Write-Host "To test build info, run:"
Write-Host ".\shortener.exe"
Write-Host "or"
Write-Host ".\api.exe" 