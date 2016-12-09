cd $PSScriptRoot/scripts/windows-ci

echo $args.Count
if ($args.Count -eq 0) {
    ./ci
}
else {
    Invoke-Expression ./$args
}

cd $PSScriptRoot