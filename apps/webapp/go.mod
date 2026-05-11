module github.com/base/base-microservice/apps/webapp

go 1.25.0

require (
	connectrpc.com/connect v1.18.1
	github.com/base/base-microservice/gen v0.0.0
	github.com/base/base-microservice/pkg/actor v0.0.0
	github.com/base/base-microservice/pkg/auth v0.0.0
	github.com/base/base-microservice/pkg/config v0.0.0
	github.com/base/base-microservice/pkg/obs v0.0.0
	github.com/rs/zerolog v1.33.0
	golang.org/x/net v0.33.0
	golang.org/x/sync v0.10.0
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260415201107-50325440f8f2.1 // indirect
	github.com/alecthomas/kong v1.10.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.68.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/base/base-microservice/gen => ../../gen
	github.com/base/base-microservice/pkg/actor => ../../pkg/actor
	github.com/base/base-microservice/pkg/auth => ../../pkg/auth
	github.com/base/base-microservice/pkg/config => ../../pkg/config
	github.com/base/base-microservice/pkg/obs => ../../pkg/obs
)
