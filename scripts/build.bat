set GOOS=wasip1
set GOARCH=wasm
go build -o ../dist/morphediff-psql-v1.0.0.wasm ../cmd/plugin/main.go