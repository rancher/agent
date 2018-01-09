param(
    [Parameter(ValueFromPipeline=$true)]
    [string]$inputStr
    )
    $strs=$inputStr.Split(",")
    $RegisterUrl=$strs[0].Trim("`"")
$rancherAgentService=get-service rancher-agent -ErrorAction Ignore
if($rancherAgentService -ne $null){
    & 'C:\Program Files\rancher\agent.exe' --unregister-service
}
& 'C:\Program Files\rancher\agent.exe' --register-service $RegisterUrl

start-service rancher-agent
write-host "start agent success"