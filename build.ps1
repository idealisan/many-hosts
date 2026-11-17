# Windows 构建：无控制台窗口 + 请求管理员权限的 manifest。
$env:GOTOOLCHAIN = 'auto'
$env:GOPROXY = 'https://goproxy.cn,direct'
$env:GODEBUG = 'http2client=0'
$env:CGO_ENABLED = '0'

$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
$source = "resources\manifest\admin_$arch.syso"
$target = "manifest_windows_$arch.syso"

# syso 只在构建期间进入包目录，避免 go test 也要求提权。
Move-Item -Path $source -Destination $target -Force
try {
    go build -ldflags "-H=windowsgui" -o ManyHosts.exe .
}
finally {
    if (Test-Path $target) {
        Move-Item -Path $target -Destination $source -Force
    }
}
