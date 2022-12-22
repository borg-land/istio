// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package istioagent

import (
	"crypto/tls"
	"errors"
	"fmt"

	"google.golang.org/grpc"

	istiogrpc "istio.io/istio/pilot/pkg/grpc"
)

func (p *XdsProxy) buildSPIREIstiodClientDialOpts(sa *Agent) ([]grpc.DialOption, error) {
	tlsOpts, err := p.getTLSOptions(sa)
	if err != nil {
		return nil, fmt.Errorf("failed to get TLS options to talk to upstream: %v", err)
	}

	tlsOpts.GetClientCertificate = func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
		workloadCert := p.spireClient.GetWorkloadCert()
		if workloadCert == nil {
			return nil, errors.New("workload certificate not available via SPIRE")
		}
		return workloadCert, nil
	}

	options, err := istiogrpc.ClientOptions(nil, tlsOpts)
	if err != nil {
		return nil, err
	}

	return options, nil
}
