Write-Host "Running CI"

cd $PSScriptRoot

./build
./test
./validate
./package