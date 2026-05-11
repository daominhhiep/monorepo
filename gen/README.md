# gen/

Generated Go code from `proto/`. Do not edit by hand.

Run `make proto` (or `buf generate`) from repo root to regenerate.

The folder structure mirrors the proto packages:

- `gen/user/` ← `proto/user/*.proto`
- `gen/apps/webapp/v1/` ← `proto/apps/webapp/v1/api.proto`

Each package emits:

- `<file>.pb.go` (messages)
- `<file>_grpc.pb.go` (gRPC server/client — kept for compat)
- `<service>connect/<file>.connect.go` (Connect-RPC server/client)
