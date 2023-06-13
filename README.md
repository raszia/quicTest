# quicTest
test for https://github.com/quic-go/quic-go/issues/3883


go build *.go

./main

go tool pprof -http "127.0.0.1:6060" <http://127.0.0.1:9911/debug/pprof/heap>
