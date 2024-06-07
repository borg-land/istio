// Copyright Solo.io, Inc
//
// Licensed under a Solo commercial license, not Apache License, Version 2 or any other variant

package endpoints

import (
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/config/host"
	"istio.io/istio/pkg/slices"
)

// For services that have a waypoint, we want to send to the waypoints rather than the service endpoints.
// Lookup the
func (b *EndpointBuilder) findServiceWaypoint(endpointIndex *model.EndpointIndex) ([]*model.IstioEndpoint, bool) {
	if b.proxy.IsWaypointProxy() {
		// waypoints don't chain to each-other
		return nil, false
	}
	if b.nodeType == model.Router {
		// Ingress behaves differently, must explicitly opt in
		if b.service.Attributes.Labels["istio.io/ingress-use-waypoint"] != "true" {
			return nil, false
		}
	}
	if b.service.GetAddressForProxy(b.proxy) == constants.UnspecifiedIP {
		// No VIP, so skip this. Currently, waypoints can only accept VIP traffic
		return nil, false
	}

	svcs := b.push.ServicesWithWaypoint(b.service.Attributes.Namespace + "/" + string(b.hostname))
	if len(svcs) == 0 {
		// Service isn't captured by a waypoint
		return nil, false
	}
	if len(svcs) > 1 {
		log.Warnf("unexpected multiple waypoint services for %v", b.clusterName)
	}
	svc := svcs[0]
	waypointClusterName := model.BuildSubsetKey(
		model.TrafficDirectionOutbound,
		"",
		host.Name(svc.WaypointHostname),
		int(svc.Service.GetWaypoint().GetHboneMtlsPort()),
	)
	endpointBuilder := NewEndpointBuilder(waypointClusterName, b.proxy, b.push)
	waypointEndpoints, _ := endpointBuilder.snapshotEndpointsForPort(endpointIndex)
	return waypointEndpoints, true
}

func (b *EndpointBuilder) snapshotEndpointsForPort(endpointIndex *model.EndpointIndex) ([]*model.IstioEndpoint, bool) {
	svcPort := b.servicePort(b.port)
	if svcPort == nil {
		return nil, false
	}
	svcEps := b.snapshotShards(endpointIndex)
	svcEps = slices.FilterInPlace(svcEps, func(ep *model.IstioEndpoint) bool {
		// filter out endpoints that don't match the service port
		if svcPort.Name != ep.ServicePortName {
			return false
		}
		// filter out endpoints that don't match the subset
		if !b.subsetLabels.SubsetOf(ep.Labels) {
			return false
		}
		return true
	})
	return svcEps, true
}
