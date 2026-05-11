// Build all container images via `docker buildx bake`.
//   docker buildx bake services  → backend images
//   docker buildx bake apis      → BFF images
//   docker buildx bake webs      → SPA images

variable "REGISTRY" { default = "base" }
variable "TAG"      { default = "dev" }
variable "VERSION"  { default = "dev" }
variable "COMMIT"   { default = "unknown" }
variable "PLATFORMS" { default = ["linux/amd64"] }

group "default" {
  targets = ["services", "apis", "webs"]
}

group "services" { targets = ["user"] }
group "apis"     { targets = ["webapp-bff"] }
group "webs"     { targets = ["webapp-web"] }

target "_go" {
  context    = "."
  dockerfile = "build/go.Dockerfile"
  platforms  = PLATFORMS
  args = {
    VERSION = VERSION
    COMMIT  = COMMIT
  }
}

target "_web" {
  context    = "."
  dockerfile = "build/web.Dockerfile"
  platforms  = PLATFORMS
}

target "user" {
  inherits = ["_go"]
  args = {
    BIN_PATH = "./services/user/cmd/user"
    BIN_NAME = "user"
  }
  tags       = ["${REGISTRY}/svc/user:${TAG}"]
  cache-from = ["type=gha,scope=user"]
  cache-to   = ["type=gha,scope=user,mode=max"]
}

target "webapp-bff" {
  inherits = ["_go"]
  args = {
    BIN_PATH = "./apps/webapp/cmd/webapp"
    BIN_NAME = "webapp"
  }
  tags       = ["${REGISTRY}/apps/webapp/api:${TAG}"]
  cache-from = ["type=gha,scope=webapp-bff"]
  cache-to   = ["type=gha,scope=webapp-bff,mode=max"]
}

target "webapp-web" {
  inherits = ["_web"]
  args = {
    WEB_PKG = "@base/webapp"
  }
  tags       = ["${REGISTRY}/apps/webapp/web:${TAG}"]
  cache-from = ["type=gha,scope=webapp-web"]
  cache-to   = ["type=gha,scope=webapp-web,mode=max"]
}
