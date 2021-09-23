module etcd-example

go 1.13

require (
	github.com/coreos/etcd v3.3.20+incompatible
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.4.1
	github.com/google/uuid v1.1.1 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/prometheus/client_golang v1.6.0
	go.etcd.io/etcd v3.3.20+incompatible
	go.uber.org/zap v1.15.0 // indirect
	golang.org/x/net v0.0.0-20200506145744-7e3656a0809f // indirect
	golang.org/x/sys v0.0.0-20200511232937-7e40ca221e25 // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20200511104702-f5ebc3bea380 // indirect
	google.golang.org/grpc v1.29.1
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0
replace github.com/coreos/bbolt v1.3.4 => go.etcd.io/bbolt v1.3.4
