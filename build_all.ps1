param(
  [string]$GoArch = "amd64"
)

$ErrorActionPreference = "Stop"

$root = $PSScriptRoot
$domains = @("users", "login", "donation", "pix", "contact")

$env:GO111MODULE = "on"
$env:GOOS = "linux"
$env:GOARCH = $GoArch
$env:CGO_ENABLED = "0"

foreach ($d in $domains) {
  $path = Join-Path $root $d
  Write-Host "Building $d..."
  Push-Location $path
  try {
    go build -o bootstrap .
    if (Test-Path -LiteralPath "lambda.zip") { Remove-Item -Force "lambda.zip" }
    Compress-Archive -Path "bootstrap" -DestinationPath "lambda.zip" -Force
  } finally {
    Pop-Location
  }
}

Write-Host "Done. Zips created in each domain folder." 
