#!/usr/bin/env bash
set -euo pipefail

CLUSTER="${CLUSTER:-base}"
k3d cluster delete "${CLUSTER}"
