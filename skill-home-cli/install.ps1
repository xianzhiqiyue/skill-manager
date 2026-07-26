[CmdletBinding()]
param(
    [string]$Version = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$BinaryName = "skill-home.exe"
$DefaultInstallDir = Join-Path $env:LOCALAPPDATA "Programs\skill-home"
$InstallDir = if ($env:SKILL_HOME_INSTALL_DIR) { $env:SKILL_HOME_INSTALL_DIR } else { $DefaultInstallDir }
$DefaultReleasesBaseUrl = "__SKILL_HOME_RELEASES_BASE_URL__"
if ($DefaultReleasesBaseUrl -eq "__SKILL_HOME_RELEASES_BASE_URL__") {
    $DefaultReleasesBaseUrl = "https://soulstore.ciqtek.com/skill-home/releases"
}
$ReleasesBaseUrl = if ($env:SKILL_HOME_RELEASES_BASE_URL) {
    $env:SKILL_HOME_RELEASES_BASE_URL.TrimEnd("/")
} else {
    $DefaultReleasesBaseUrl.TrimEnd("/")
}

function Get-NormalizedVersion {
    param(
        [string]$RequestedVersion,
        [string]$TemporaryDirectory
    )

    $resolvedVersion = $RequestedVersion.Trim()
    if (-not $resolvedVersion) {
        Write-Host "正在获取最新版本..."
        $latestPath = Join-Path $TemporaryDirectory "latest.json"
        Invoke-WebRequest -UseBasicParsing -Uri "$ReleasesBaseUrl/latest.json" -OutFile $latestPath
        $latest = Get-Content -Raw -Path $latestPath | ConvertFrom-Json
        $resolvedVersion = [string]$latest.tag_name
        if (-not $resolvedVersion.Trim()) {
            throw "latest.json 中没有可用的 tag_name"
        }
    }

    if (-not $resolvedVersion.StartsWith("v")) {
        $resolvedVersion = "v$resolvedVersion"
    }

    return $resolvedVersion
}

function Assert-SupportedArchitecture {
    $architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
    if ($architecture -ne "X64") {
        throw "当前 Windows 架构为 $architecture；Skill Home CLI 目前只提供 Windows x64 安装包。"
    }
}

function Add-InstallDirectoryToUserPath {
    param([string]$Directory)

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $entries = @($userPath -split ";" | Where-Object { $_ })
    if ($entries -notcontains $Directory) {
        $updatedPath = (@($entries) + $Directory) -join ";"
        [Environment]::SetEnvironmentVariable("Path", $updatedPath, "User")
        Write-Host "已将 $Directory 加入当前用户 PATH。"
    }

    $processEntries = @($env:Path -split ";" | Where-Object { $_ })
    if ($processEntries -notcontains $Directory) {
        $env:Path = "$Directory;$env:Path"
    }
}

function Install-SkillHome {
    Write-Host "================================"
    Write-Host "  skill-home CLI Windows 安装脚本"
    Write-Host "================================"
    Write-Host ""

    Assert-SupportedArchitecture

    $temporaryDirectory = Join-Path ([IO.Path]::GetTempPath()) ("skill-home-" + [guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $temporaryDirectory -Force | Out-Null

    try {
        $normalizedVersion = Get-NormalizedVersion -RequestedVersion $Version -TemporaryDirectory $temporaryDirectory
        $assetName = "skill-home-windows-amd64.zip"
        $archivePath = Join-Path $temporaryDirectory $assetName
        $checksumsPath = Join-Path $temporaryDirectory "checksums.txt"
        $extractPath = Join-Path $temporaryDirectory "extract"
        $versionBaseUrl = "$ReleasesBaseUrl/$normalizedVersion"

        Write-Host "版本: $normalizedVersion"
        Write-Host "平台: windows-amd64"
        Write-Host "安装目录: $InstallDir"
        Write-Host "Skill Home 发布源: $ReleasesBaseUrl"
        Write-Host ""
        Write-Host "正在下载发布包..."

        Invoke-WebRequest -UseBasicParsing -Uri "$versionBaseUrl/$assetName" -OutFile $archivePath
        Invoke-WebRequest -UseBasicParsing -Uri "$versionBaseUrl/checksums.txt" -OutFile $checksumsPath

        $checksumLine = Get-Content -Path $checksumsPath |
            Where-Object { $_ -match ("\s(?:\./)?" + [regex]::Escape($assetName) + "$") } |
            Select-Object -First 1
        if (-not $checksumLine) {
            throw "校验文件中没有找到 $assetName"
        }

        $expectedChecksum = ($checksumLine -split "\s+")[0].ToLowerInvariant()
        $actualChecksum = (Get-FileHash -Algorithm SHA256 -Path $archivePath).Hash.ToLowerInvariant()
        if ($expectedChecksum -ne $actualChecksum) {
            throw "checksum 校验失败: $assetName"
        }

        Write-Host "正在解压..."
        Expand-Archive -LiteralPath $archivePath -DestinationPath $extractPath -Force

        $binaryPath = Join-Path $extractPath $BinaryName
        if (-not (Test-Path -LiteralPath $binaryPath -PathType Leaf)) {
            throw "安装包内未找到 $BinaryName"
        }

        Write-Host "正在安装..."
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        Copy-Item -LiteralPath $binaryPath -Destination (Join-Path $InstallDir $BinaryName) -Force
        Add-InstallDirectoryToUserPath -Directory $InstallDir

        Write-Host ""
        Write-Host "安装完成: $(Join-Path $InstallDir $BinaryName)"
        Write-Host "请运行 'skill-home doctor' 检查注册中心和本地 Agent 路径。"
        Write-Host "如果当前终端仍找不到 skill-home，请重新打开 PowerShell。"
    } finally {
        Remove-Item -LiteralPath $temporaryDirectory -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Install-SkillHome
