// Copyright Solo.io, Inc
//
// Licensed under a Solo commercial license, not Apache License, Version 2 or any other variant

package features

var EnableWaypointInterop = registerAmbient("ENABLE_WAYPOINT_INTEROP", true, false,
	"If true, sidecars will short-circuit all processing and connect directly to a waypoint if the destination service has a waypoint. "+
		"Gateways will do the same if the 'istio.io/ingress-use-waypoint' label is set on the Service")
