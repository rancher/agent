Write-Host "Validating"

cd $PSScriptRoot/../..

$package = Get-ChildItem . -Name *.go -Recurse | Split-Path -Parent | Sort-Object -Unique | Select-String -Pattern "(^\.$|.git|.trash-cache|vendor|bin)" -NotMatch | ForEach-Object { "github.com/rancher/agent/$_" }

Write-Host "Running: go vet"
go vet $package

Write-Host "Running: golint"
foreach ($p in $package) {
    if ( $(golint $p | Select-String -Pattern "should have comment.*or be unexported" -NotMatch | Write-Error).length -ne 0) {
        $failed = true
    }
}
if ($failed) {
    exit 1
}
Write-Host "Running: go fmt"
if ($(go fmt $package | Write-Error).length -ne 0) {
    exit 1
}

cd $PSScriptRoot