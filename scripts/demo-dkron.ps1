param(
    [switch]$Failover,
    [switch]$UnitTests,
    [switch]$SkipSeed,
    [int]$SampleRuns = 2,
    [string]$Root = "http://localhost:8080"
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$DkronSourcePath = Join-Path $ProjectRoot "dkron"

function Write-Step($text) {
    Write-Host ""
    Write-Host "== $text ==" -ForegroundColor Cyan
}

function Invoke-Json($Method, $Uri, $Body = $null) {
    if ($null -eq $Body) {
        return Invoke-RestMethod -Method $Method -Uri $Uri
    }

    return Invoke-RestMethod -Method $Method -Uri $Uri `
        -ContentType "application/json" `
        -Body ($Body | ConvertTo-Json -Depth 10)
}

function Show-Json($value) {
    $value | ConvertTo-Json -Depth 10
}

function Get-LeaderName($leader) {
    if ($leader.Name) { return $leader.Name }
    if ($leader.Member -and $leader.Member.Name) { return $leader.Member.Name }
    return $null
}

function New-DemoJob($Name, $DisplayName, $Schedule, $Command, $Owner, $Retries = 0) {
    return @{
        name = $Name
        displayname = $DisplayName
        schedule = $Schedule
        owner = $Owner
        owner_email = "$Owner@example.local"
        retries = $Retries
        concurrency = "allow"
        metadata = @{
            course = "distributed-applications"
            demo = "midterm"
            scenario = $Name
        }
        executor = "shell"
        executor_config = @{
            command = $Command
            shell = "true"
        }
    }
}

function Run-DemoJob($Name) {
    Invoke-Json POST "$Root/v1/jobs/$Name/run" | Out-Null
    Write-Host "Da yeu cau chay: $Name"
}

function Show-ExecutionSummary($Name, $Limit = 3) {
    $items = @(Invoke-Json GET "$Root/v1/jobs/$Name/executions" | ForEach-Object { $_ })
    $success = @($items | Where-Object { $_.success -eq $true }).Count
    $failed = @($items | Where-Object { $_.success -ne $true }).Count
    $nodes = @($items | ForEach-Object { $_.node_name } | Sort-Object -Unique) -join ", "

    Write-Host "Job: $Name"
    Write-Host "Tong executions: $($items.Count), thanh cong: $success, that bai: $failed, nodes: $nodes"
    $items |
        Select-Object -First $Limit id, success, node_name, started_at, finished_at, output |
        ConvertTo-Json -Depth 5
}

Write-Step "1. Kiem tra Docker Compose"
docker compose ps

Write-Step "2. Kiem tra health, members va leader"
$health = Invoke-Json GET "$Root/health"
Write-Host "Health:"
Show-Json $health

$members = Invoke-Json GET "$Root/v1/members"
Write-Host "Members:"
Show-Json $members

$leader = Invoke-Json GET "$Root/v1/leader"
$leaderName = Get-LeaderName $leader
Write-Host "Leader:"
Show-Json $leader

Write-Step "3. Tao job demo va chay ngay"
$job = New-DemoJob `
    "demo_analytics_job" `
    "Analytics baseline job" `
    "@every 1h" `
    "echo demo-run && date" `
    "analytics-team"

Invoke-Json POST "$Root/v1/jobs" $job | Out-Null
Write-Host "Da tao/cap nhat job demo_analytics_job"

Run-DemoJob "demo_analytics_job"
Start-Sleep -Seconds 5

if (-not $SkipSeed) {
    Write-Step "3b. Seed nhieu job mau de demo UI va Analytics"
    $sampleJobs = @(
        (New-DemoJob `
            "demo_fast_success" `
            "Fast success task" `
            "@every 2h" `
            "echo fast-success && date" `
            "ops-team"),
        (New-DemoJob `
            "demo_slow_success" `
            "Slow success task" `
            "@every 3h" `
            "echo slow-start && sleep 2 && echo slow-done" `
            "data-team"),
        (New-DemoJob `
            "demo_backup_pipeline" `
            "Backup pipeline" `
            "@every 5h" `
            "echo backup-start && sleep 1 && echo backup-ok" `
            "platform-team"),
        (New-DemoJob `
            "demo_report_generator" `
            "Report generator" `
            "@every 7h" `
            "echo report-generated && date" `
            "business-team"),
        (New-DemoJob `
            "demo_cleanup_task" `
            "Cleanup task" `
            "@every 11h" `
            "echo cleanup-temp-files && sleep 1 && echo cleanup-ok" `
            "ops-team"),
        (New-DemoJob `
            "demo_intentional_failure" `
            "Intentional failure task" `
            "@every 13h" `
            "echo intentional-failure && exit 1" `
            "qa-team" `
            1)
    )

    foreach ($sampleJob in $sampleJobs) {
        Invoke-Json POST "$Root/v1/jobs" $sampleJob | Out-Null
        Write-Host "Da tao/cap nhat job mau: $($sampleJob.name)"
    }

    for ($round = 1; $round -le $SampleRuns; $round++) {
        Write-Host ""
        Write-Host "Vong chay mau $round/$SampleRuns"
        foreach ($sampleJob in $sampleJobs) {
            Run-DemoJob $sampleJob.name
            Start-Sleep -Milliseconds 700
        }
    }

    Write-Host "Cho cac execution mau hoan tat..."
    Start-Sleep -Seconds 8
}

Write-Step "4. Xem executions cua job demo"
Show-ExecutionSummary "demo_analytics_job"

if (-not $SkipSeed) {
    Write-Step "4b. Tom tat executions cua cac job mau"
    foreach ($name in @(
        "demo_fast_success",
        "demo_slow_success",
        "demo_backup_pipeline",
        "demo_report_generator",
        "demo_cleanup_task",
        "demo_intentional_failure"
    )) {
        Write-Host ""
        try {
            Show-ExecutionSummary $name
        } catch {
            Write-Host "Khong doc duoc executions cua ${name}: $($_.Exception.Message)" -ForegroundColor Yellow
        }
    }
}

Write-Step "5. Goi Analytics API moi"
$analytics = Invoke-Json GET "$Root/v1/analytics"
Show-Json $analytics

Write-Step "6. Kiem tra Analytics API qua 3 node"
foreach ($port in 8080, 8081, 8082) {
    $url = "http://localhost:$port/v1/analytics"
    try {
        Write-Host "Node port ${port}:"
        Show-Json (Invoke-Json GET $url)
    } catch {
        Write-Host "Khong goi duoc ${url}: $($_.Exception.Message)" -ForegroundColor Yellow
    }
}

if ($UnitTests) {
    Write-Step "7. Chay test tu dong cho 2 tinh nang"
    docker run --rm -v "${DkronSourcePath}:/app" -w /app giuaki-dkron-server-1:latest go test ./dkron -run TestAPIAnalytics -count=1
    docker run --rm -v "${DkronSourcePath}:/app" -w /app giuaki-dkron-server-1:latest go test ./plugin/webhook -count=1
}


if ($Failover) {
    Write-Step "8. Demo failover leader"
    if (-not $leaderName) {
        Write-Host "Khong xac dinh duoc leader name tu API /v1/leader" -ForegroundColor Yellow
    } else {
        $containerMap = @{
            "server1" = "giuaki-dkron-server-1-1"
            "server2" = "giuaki-dkron-server-2-1"
            "server3" = "giuaki-dkron-server-3-1"
        }
        $container = $containerMap[$leaderName]
        if (-not $container) {
            Write-Host "Khong map duoc leader $leaderName sang container" -ForegroundColor Yellow
        } else {
            Write-Host "Leader hien tai: $leaderName, container: $container"
            docker stop $container | Out-Null
            Write-Host "Da stop leader. Cho Raft bau leader moi..."
            Start-Sleep -Seconds 8

            foreach ($port in 8080, 8081, 8082) {
                $url = "http://localhost:$port/v1/leader"
                try {
                    Write-Host "Leader nhin tu port ${port}:"
                    Show-Json (Invoke-Json GET $url)
                } catch {
                    Write-Host "Port ${port} khong phan hoi: $($_.Exception.Message)" -ForegroundColor Yellow
                }
            }

            Write-Host ""
            Write-Host "Khoi dong lai leader cu de tra cum ve trang thai 3 node:"
            Write-Host "docker start $container" -ForegroundColor Yellow
        }
    }
}

Write-Step "Hoan tat demo"
Write-Host "UI: http://localhost:8080/ui"
Write-Host "Analytics API: $Root/v1/analytics"
