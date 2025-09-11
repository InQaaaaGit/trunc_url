# PowerShell script for generating Go code from proto file

Write-Host "Generating Go code from proto file..." -ForegroundColor Green

# Create proto directory if it doesn't exist
if (!(Test-Path "proto")) {
    New-Item -ItemType Directory -Path "proto"
}

# Generate Go code from proto file
protoc --go_out=. --go_opt=paths=source_relative `
       --go-grpc_out=. --go-grpc_opt=paths=source_relative `
       proto/url_service.proto

Write-Host "Generation completed!" -ForegroundColor Green 