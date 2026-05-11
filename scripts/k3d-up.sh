#!/usr/bin/env bash
# Create a local k3d cluster with ingress-nginx on host ports 80/443.
set -euo pipefail

CLUSTER="${CLUSTER:-base}"

if k3d cluster list "${CLUSTER}" >/dev/null 2>&1; then
  echo "→ Cluster '${CLUSTER}' already exists. Use 'make k3d-down' to remove."
  exit 0
fi

echo "→ Creating k3d cluster '${CLUSTER}' (1 server + 2 agents)..."
k3d cluster create "${CLUSTER}" \
  --agents 2 \
  --port "80:80@loadbalancer" \
  --port "443:443@loadbalancer" \
  --k3s-arg "--disable=traefik@server:0"

echo "→ Installing ingress-nginx (manifests reference ingressClassName: nginx)..."
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.3/deploy/static/provider/cloud/deploy.yaml
kubectl -n ingress-nginx wait --for=condition=Available deploy/ingress-nginx-controller --timeout=300s

echo
echo "✓ Cluster ready. kubectl context: k3d-${CLUSTER}"
echo "  Add to /etc/hosts so the webapp ingress resolves locally:"
echo "    127.0.0.1  app.example.com"
