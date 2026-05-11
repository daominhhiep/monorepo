#!/usr/bin/env bash
# Install Argo CD into the current cluster and register the monorepo + dev app.
#
# Required env:
#   GITHUB_USERNAME  – your GitHub login (e.g. daominhhiep)
#   GITHUB_TOKEN     – PAT with scopes: repo, read:packages
set -euo pipefail

ARGOCD_NS="argocd"
APP_NS="base-dev"
ARGOCD_VERSION="${ARGOCD_VERSION:-v2.13.0}"

: "${GITHUB_USERNAME:?Set GITHUB_USERNAME (your GitHub login)}"
: "${GITHUB_TOKEN:?Set GITHUB_TOKEN (PAT with repo + read:packages scopes)}"

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "→ Installing Argo CD ${ARGOCD_VERSION}..."
kubectl create namespace "${ARGOCD_NS}" --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -n "${ARGOCD_NS}" \
  -f "https://raw.githubusercontent.com/argoproj/argo-cd/${ARGOCD_VERSION}/manifests/install.yaml"
kubectl -n "${ARGOCD_NS}" rollout status deploy/argocd-server --timeout=300s
kubectl -n "${ARGOCD_NS}" rollout status deploy/argocd-repo-server --timeout=300s

echo "→ Registering private monorepo with Argo CD..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: repo-monorepo
  namespace: ${ARGOCD_NS}
  labels:
    argocd.argoproj.io/secret-type: repository
stringData:
  type: git
  url: https://github.com/daominhhiep/monorepo.git
  username: ${GITHUB_USERNAME}
  password: ${GITHUB_TOKEN}
EOF

echo "→ Preparing '${APP_NS}' namespace with GHCR pull secret..."
kubectl create namespace "${APP_NS}" --dry-run=client -o yaml | kubectl apply -f -

kubectl -n "${APP_NS}" create secret docker-registry ghcr-pull \
  --docker-server=ghcr.io \
  --docker-username="${GITHUB_USERNAME}" \
  --docker-password="${GITHUB_TOKEN}" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n "${APP_NS}" patch serviceaccount default \
  --type merge \
  -p '{"imagePullSecrets":[{"name":"ghcr-pull"}]}'

echo "→ Creating Argo CD Application..."
kubectl apply -f "${REPO_DIR}/deploy/argocd/app-dev.yaml"

echo
echo "✓ Done."
echo
echo "Open Argo CD UI:"
echo "    make argocd-ui      # → https://localhost:8080"
echo "    make argocd-pass    # → admin password"
