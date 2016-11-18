Write-Host "Testing"

cd $PSScriptRoot/../..

$package = Get-ChildItem . -Name *.go -Recurse | Split-Path -Parent | Sort-Object -Unique | Select-String -Pattern "(^\.$|.git|.trash-cache|vendor|bin|hostapi)" -NotMatch | ForEach-Object { "github.com/rancher/agent/$_" }

go test -race -cover -timeout=3m -tags=test $package

cd $PSScriptRoot