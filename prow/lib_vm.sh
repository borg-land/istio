#!/bin/bash

# Copyright 2018 Istio Authors
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

build_images_for_vm_test() {
  # Build just the images needed for VM tests
  targets="docker.pilot docker.proxyv2 docker.install-cni docker.ztunnel docker.istioctl"

  # Integration tests are always running on local architecture (no cross compiling), so find out what that is.
  arch="linux/amd64"
  if [[ "$(uname -m)" == "aarch64" ]]; then
      arch="linux/arm64"
  fi
  if [[ "${VARIANT:-default}" == "distroless" ]]; then
    DOCKER_ARCHITECTURES="${arch}" DOCKER_BUILD_VARIANTS="distroless" DOCKER_TARGETS="${targets}" make dockerx.pushx
  else
   DOCKER_ARCHITECTURES="${arch}"  DOCKER_BUILD_VARIANTS="${VARIANT:-default}" DOCKER_TARGETS="${targets}" make dockerx.pushx
  fi

  make deb ambient_deb spire-agent_deb spire-server_deb
}