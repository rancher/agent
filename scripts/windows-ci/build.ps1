Write-Host "Building"

cd $PSScriptRoot/../..

New-Item bin -ItemType Directory -Force
go build -o bin/agent

cd $PSScriptRoot