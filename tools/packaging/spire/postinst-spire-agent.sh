#!/bin/bash
#
# Copyright 2017, 2018 Istio Authors. All Rights Reserved.
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
#
################################################################################

set -e

umask 022

if ! getent passwd istio-proxy >/dev/null; then
    if command -v useradd >/dev/null; then
        groupadd --system istio-proxy
        useradd --system --gid istio-proxy --home-dir /var/lib/spire istio-proxy
    else
        addgroup --system istio-proxy
        adduser --system --group --home /var/lib/spire istio-proxy
    fi
fi

if [ ! -e /etc/spire ]; then
   # Backward compat.
   ln -s /var/lib/spire /etc/spire
fi

mkdir -p /var/lib/spire/certs
mkdir -p /var/run/spire/
mkdir -p /var/log/spire

chown -R istio-proxy.istio-proxy /var/lib/spire/ /var/run/spire/ /var/log/spire/
chmod o+rx /usr/local/bin/spire-agent
chmod 2755 /usr/local/bin/spire-agent
