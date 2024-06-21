#!/bin/bash

# Copyright Istio Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

WD=$(dirname "$0")
WD=$(cd "$WD"; pwd)
ROOT=$(dirname "$WD")

set -eux

# shellcheck source=prow/lib.sh
source "${ROOT}/prow/lib.sh"

setup_gcloud_credentials

# Old prow image does not set this, so needed explicitly here as this is not called through make
export GO111MODULE=on

DOCKER_HUB=${DOCKER_HUB:-gcr.io/istio-testing}
GCS_BUCKET=${GCS_BUCKET:-istio-build/dev}

# Enable emulation required for cross compiling a few images (VMs)
# **SOLO** Leave this as hardcoded since we do not currently mirror this -Daniel
docker run --rm --privileged "gcr.io/istio-testing/qemu-user-static" --reset -p yes
export ISTIO_DOCKER_QEMU=true

# Use a pinned version in case breaking changes are needed
BUILDER_SHA=69dba7da9d0b0404c90202588468ec5144c88b15

# Reference to the next minor version of Istio
# This will create a version like 1.4-alpha.sha
NEXT_VERSION=$(cat "${ROOT}/VERSION")
TAG=$(git rev-parse HEAD)
VERSION="${NEXT_VERSION}-alpha.${TAG}"

# In CI we want to store the outputs to artifacts, which will preserve the build
# If not specified, we can just create a temporary directory
WORK_DIR="$(mktemp -d)/build"
mkdir -p "${WORK_DIR}"

MANIFEST=$(cat <<EOF
version: ${VERSION}
docker: ${DOCKER_HUB}
directory: ${WORK_DIR}
ignoreVulnerability: true
dependencies:
${DEPENDENCIES:-$(cat <<EOD
  istio:
    localpath: ${ROOT}
  api:
    git: https://github.com/istio/api
    auto: modules
  proxy:
    git: https://github.com/istio/proxy
    auto: deps
  client-go:
    git: https://github.com/istio/client-go
    branch: release-1.23
  test-infra:
    git: https://github.com/istio/test-infra
    branch: master
  tools:
    git: https://github.com/istio/tools
    branch: release-1.23
  release-builder:
    git: https://github.com/istio/release-builder
    sha: ${BUILDER_SHA}
  ztunnel:
    git: git@github.com:solo-io/ztunnel.git
    auto: deps
architectures: [linux/amd64, linux/arm64]
EOD
)}
dashboards:
  istio-mesh-dashboard: 7639
  istio-performance-dashboard: 11829
  istio-service-dashboard: 7636
  istio-workload-dashboard: 7630
  pilot-dashboard: 7645
  istio-extension-dashboard: 13277
  ztunnel-dashboard: 21306
${PROXY_OVERRIDE:-}
EOF
)

# "Temporary" hacks
export PATH=${GOPATH}/bin:${PATH}

go install "istio.io/release-builder@${BUILDER_SHA}"

release-builder build --manifest <(echo "${MANIFEST}")

## ilrudie - moving to a model that's based more closely on the normal release builder... all extra builds/packages/copies will be handled in make
## TODO - cleanup once confirmed unneccesary.
# # Build our VM packages and place in the out directory
# function cp_to_output() {
#   local repo=$1
#   local arch=$2
#   local filename=$3
#   local output=$4

#   local source="${WORK_DIR}/work/src/istio.io/${repo}/out/linux_${arch}/release/${filename}"
#   local output="${WORK_DIR}/out/${output}"

#   if [[ ! -f "$source" ]]; then
#     echo "ERROR: $source does not exist"
#     exit 1
#   fi

#   cp "$source" "$output"
# }

# # This only builds deb and CentOS 8+ RPMs, we don't seem to be outputting the CentOS 7 rpm anywhere
# pushd "${WORK_DIR}/work/src/istio.io/istio"
# # Build istio-ambient
# make ambient_deb/fpm
# cp_to_output istio amd64 istio-ambient.deb deb/istio-ambient.deb
# #cp_to_output istio arm64 istio-ambient-arm64.deb deb/istio-ambient-arm64.deb
# make ambient_rpm/fpm
# cp_to_output istio amd64 istio-ambient.rpm rpm/istio-ambient.rpm
# #cp_to_output istio arm64 istio-ambient-arm64.rpm rpm/istio-ambient-arm64.rpm

# # Build spire-agent
# make spire-agent_deb/fpm
# cp_to_output istio amd64 spire-agent.deb deb/spire-agent.deb
# #cp_to_output istio arm64 spire-agent-arm64.deb deb/spire-agent-arm64.deb
# make spire-agent_rpm/fpm
# cp_to_output istio amd64 spire-agent.rpm rpm/spire-agent.rpm
# #cp_to_output istio arm64 spire-agent-arm64.rpm rpm/spire-agent-arm64.rpm

# # Build spire-server
# make spire-server_deb/fpm
# cp_to_output istio amd64 spire-server.deb deb/spire-server.deb
# #cp_to_output istio arm64 spire-server-arm64.deb deb/spire-server-arm64.deb
# make spire-server_rpm/fpm
# cp_to_output istio amd64 spire-server.rpm rpm/spire-server.rpm
# #cp_to_output istio arm64 spire-server-arm64.rpm rpm/spire-server-arm64.rpm
# popd

release-builder validate --release "${WORK_DIR}/out"

if [[ -z "${DRY_RUN:-}" ]]; then
  release-builder publish --release "${WORK_DIR}/out" \
    --gcsbucket "${GCS_BUCKET}" --gcsaliases "${TAG},${NEXT_VERSION}-dev" \
    --dockerhub "${DOCKER_HUB}" --dockertags "${TAG},${VERSION},${NEXT_VERSION}-dev" \
    --helmhub "${HELM_HUB}"
fi
