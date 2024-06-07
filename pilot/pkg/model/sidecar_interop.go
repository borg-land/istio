// Copyright Solo.io, Inc
//
// Licensed under a Solo commercial license, not Apache License, Version 2 or any other variant

package model

import "istio.io/istio/pkg/workloadapi"

type ServiceWaypointInfo struct {
	Service          *workloadapi.Service
	WaypointHostname string
}

// ServicesWithWaypoint returns all services associated with any waypoint.
// Key can optionally be provided in the form 'namespace/hostname'. If unset, all are returned
func (ps *PushContext) ServicesWithWaypoint(key string) []ServiceWaypointInfo {
	return ps.ambientIndex.ServicesWithWaypoint(key)
}

func (u NoopAmbientIndexes) ServicesWithWaypoint(string) []ServiceWaypointInfo {
	return nil
}
