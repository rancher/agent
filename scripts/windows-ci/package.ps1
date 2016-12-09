Write-Host "Packaging"

cd $PSScriptRoot/../..

New-Item dist/artifacts -ItemType Directory -Force

Compress-Archive -Path ./bin/agent -DestinationPath dist/artifacts/agent.zip -Force
Write-Host Created ./dist/artifacts/agent.zip

cd $PSScriptRoot