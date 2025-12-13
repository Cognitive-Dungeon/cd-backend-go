$ErrorActionPreference = "Stop"

# --- Project ---
$APP_NAME  = "cognitive-server"
$MODULE   = "cognitive-server"
$CMD_PATH = "./cmd/server/main.go"
$BUILD_DIR = "./bin"

$BIN_NAME = "$APP_NAME.exe"
$BIN_PATH = Join-Path $BUILD_DIR $BIN_NAME

# --- Build metadata ---
$BUILD_DATE = (Get-Date).ToUniversalTime().ToString("yyyy-MM-dd")

try {
    $GIT_COMMIT = (git rev-parse --short HEAD).Trim()
} catch {
    $GIT_COMMIT = "unknown"
}

try {
    $GIT_BRANCH = (git branch --show-current).Trim()
} catch {
    $GIT_BRANCH = "unknown"
}

$BUILD_CI = if ($env:CI) { $env:CI } else { "local" }

$LDFLAGS = @(
    "-X $MODULE/internal/version.BuildDate=$BUILD_DATE"
    "-X $MODULE/internal/version.BuildCommit=$GIT_COMMIT"
    "-X $MODULE/internal/version.BuildBranch=$GIT_BRANCH"
    "-X $MODULE/internal/version.BuildCI=$BUILD_CI"
) -join " "

Write-Host "Building $APP_NAME for Windows..."
Write-Host "  Date   : $BUILD_DATE"
Write-Host "  Commit : $GIT_COMMIT"
Write-Host "  Branch : $GIT_BRANCH"
Write-Host "  CI     : $BUILD_CI"

# --- Build ---
if (!(Test-Path $BUILD_DIR)) {
    New-Item -ItemType Directory -Force -Path $BUILD_DIR | Out-Null
}

go build `
    -ldflags "$LDFLAGS" `
    -o $BIN_PATH `
    $CMD_PATH

if ($LASTEXITCODE -ne 0) {
    Write-Error "Build failed (exit code $LASTEXITCODE)"
    exit $LASTEXITCODE
}

Write-Host "Successfully built to $BIN_PATH"
