mkdir -p dists
go generate
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w"
tar czvf dists/webfs-linux.tar.gz webfs

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w"
zip dists/webfs-windows.zip webfs.exe

