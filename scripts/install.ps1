# ctx installer script for Windows
# Usage: iwr -useb https://raw.githubusercontent.com/akash202k/ctx/main/scripts/install.ps1 | iex

$ErrorActionPreference = "Stop"

$REPO = "akash202k/ctx"
$BINARY = "ctx"

Write-Host "Installing ctx..." -ForegroundColor Green

# Check if Go is installed
$goVersion = $null
try {
    $goVersion = go version 2>$null
} catch {}

if ($goVersion) {
    Write-Host "✓ Go detected, installing via 'go install'..." -ForegroundColor Green
    
    go install github.com/$REPO/cmd/ctx@latest
    
    if ($LASTEXITCODE -eq 0) {
        # Get GOPATH
        $goPath = go env GOPATH
        $ctxPath = Join-Path $goPath "bin\ctx.exe"
        
        if (Test-Path $ctxPath) {
            Write-Host "✓ ctx installed successfully!" -ForegroundColor Green
            Write-Host ""
            Write-Host "Usage:"
            Write-Host "  ctx           - Interactive wizard (default)"
            Write-Host "  ctx read      - Generate repository snapshot"
            Write-Host "  ctx edit      - Apply edits from snapshot"
            Write-Host "  ctx select    - Select relevant files for a task"
            Write-Host ""
            Write-Host "Run 'ctx --help' for more information."
            
            # Check if GOPATH\bin is in PATH
            $pathParts = $env:PATH -split ";"
            $goBinPath = Join-Path $goPath "bin"
            
            if ($pathParts -notcontains $goBinPath) {
                Write-Host ""
                Write-Host "⚠ Add Go bin to PATH for easy access:" -ForegroundColor Yellow
                Write-Host "  Run in PowerShell (Admin):"
                Write-Host "  [Environment]::SetEnvironmentVariable('PATH', `$env:PATH + ';$goBinPath', 'User')"
                Write-Host ""
                Write-Host "Or add manually via System Properties > Environment Variables"
            }
            
            exit 0
        }
    }
}

Write-Host "⚠ Go not found or installation failed." -ForegroundColor Yellow
Write-Host ""
Write-Host "Please install Go from https://golang.org/dl/"
Write-Host ""
Write-Host "After installing Go, run:"
Write-Host "  go install github.com/$REPO/cmd/ctx@latest"
Write-Host ""
Write-Host "Then add Go's bin directory to your PATH."

exit 1
