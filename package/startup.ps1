param(
    [Parameter(ValueFromPipeline=$true)]
    [string]$inputStr
    )
    $strs=$inputStr.Split(",")
    $RegisterUrl=$strs[0].Trim("`"")
    $HostLabels=$strs[1].Trim("`"")
    $AgentIp=$strs[2].Trim("`"")

if($HostLabels -ne $null){
    [System.Environment]::SetEnvironmentVariable("CATTLE_HOST_LABELS",$HostLabels,"Machine")
}
if ($AgentIp -ne $null){
    [System.Environment]::SetEnvironmentVariable("CATTLE_AGENT_IP",$AgentIp,"Machine")
}
& 'C:\Program Files\rancher\startup_per-host-subnet.ps1'
$rancherAgentService=get-service rancher-agent -ErrorAction Ignore
if($rancherAgentService -ne $null){
    & 'C:\Program Files\rancher\agent.exe' --unregister-service
}
& 'C:\Program Files\rancher\agent.exe' --register-service $RegisterUrl

start-service rancher-agent
sleep 5
start-service rancher-per-host-subnet
write-host "start agent success"