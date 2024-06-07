// Copyright Solo.io, Inc
//
// Licensed under a Solo commercial license, not Apache License, Version 2 or any other variant

// nolint: gocritic
package ambient

import (
	"net/netip"
	"strings"

	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config/schema/kind"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/maps"
	"istio.io/istio/pkg/ptr"
	"istio.io/istio/pkg/slices"
	"istio.io/istio/pkg/util/sets"
	"istio.io/istio/pkg/workloadapi"
)

type serviceEDS struct {
	ServiceKey       string
	WaypointInstance []*workloadapi.Workload
}

func (w serviceEDS) ResourceName() string {
	return w.ServiceKey
}

type workloadEDS struct {
	WorkloadKey      string
	TunnelProtocol   workloadapi.TunnelProtocol
	WaypointInstance []*workloadapi.Workload
	ServiceHostnames []string
}

func (w workloadEDS) ResourceName() string {
	return w.WorkloadKey
}

// RegisterEdsShim handles triggering xDS events when Envoy EDS needs to change.
// Most of ambient index works to build `workloadapi` types - Workload, Service, etc.
// Envoy uses a different API, with different relationships between types.
// To ensure Envoy are updated properly on changes, we compute this information.
// Currently, this is only used to trigger events.
// Ideally, the information we are using in Envoy and the event trigger are using the same data directly.
func RegisterEdsShim(
	xdsUpdater model.XDSUpdater,
	Workloads krt.Collection[model.WorkloadInfo],
	WorkloadsByServiceKey krt.Index[string, model.WorkloadInfo],
	Services krt.Collection[model.ServiceInfo],
	ServicesByAddress krt.Index[networkAddress, model.ServiceInfo],
) {
	// When sending information to a workload, we need two bits of information:
	// * Does it support tunnel? If so, we will need to use HBONE
	// * Does it have a waypoint? if so, we will need to send to the waypoint
	// Record both of these.
	// Note: currently, EDS uses the waypoint VIP for workload waypoints. This is probably something that will change.
	WorkloadEds := krt.NewCollection(
		Workloads,
		func(ctx krt.HandlerContext, wl model.WorkloadInfo) *workloadEDS {
			if wl.Waypoint == nil && wl.TunnelProtocol == workloadapi.TunnelProtocol_NONE {
				return nil
			}
			res := &workloadEDS{
				WorkloadKey:      wl.ResourceName(),
				TunnelProtocol:   wl.TunnelProtocol,
				ServiceHostnames: slices.Sort(maps.Keys(wl.Services)),
			}
			if wl.Waypoint != nil {
				wAddress := wl.Waypoint.GetAddress()
				serviceKey := networkAddress{
					network: wAddress.Network,
					ip:      mustByteIPToString(wAddress.Address),
				}
				svc := krt.FetchOne(ctx, Services, krt.FilterIndex(ServicesByAddress, serviceKey))
				if svc != nil {
					workloads := krt.Fetch(ctx, Workloads, krt.FilterIndex(WorkloadsByServiceKey, svc.ResourceName()))
					res.WaypointInstance = slices.Map(workloads, func(e model.WorkloadInfo) *workloadapi.Workload {
						return e.Workload
					})
				}
			}
			return res
		},
		krt.WithName("WorkloadEds"))
	ServiceEds := krt.NewCollection(
		Services,
		func(ctx krt.HandlerContext, svc model.ServiceInfo) *serviceEDS {
			if svc.Service.Waypoint == nil {
				return nil
			}
			wAddress := svc.Service.Waypoint.GetAddress()
			serviceKey := networkAddress{
				network: wAddress.Network,
				ip:      mustByteIPToString(wAddress.Address),
			}
			waypointSvc := krt.FetchOne(ctx, Services, krt.FilterIndex(ServicesByAddress, serviceKey))
			if waypointSvc == nil {
				return nil
			}
			workloads := krt.Fetch(ctx, Workloads, krt.FilterIndex(WorkloadsByServiceKey, waypointSvc.ResourceName()))
			return &serviceEDS{
				ServiceKey: svc.ResourceName(),
				WaypointInstance: slices.Map(workloads, func(e model.WorkloadInfo) *workloadapi.Workload {
					return e.Workload
				}),
			}
		},
		krt.WithName("ServiceEds"))
	WorkloadEds.RegisterBatch(
		PushAllXds(xdsUpdater, func(i workloadEDS, updates sets.Set[model.ConfigKey]) {
			for _, svc := range i.ServiceHostnames {
				ns, hostname, _ := strings.Cut(svc, "/")
				updates.Insert(model.ConfigKey{Kind: kind.ServiceEntry, Name: hostname, Namespace: ns})
			}
		}), false)
	ServiceEds.RegisterBatch(
		PushAllXds(xdsUpdater, func(svc serviceEDS, updates sets.Set[model.ConfigKey]) {
			ns, hostname, _ := strings.Cut(svc.ServiceKey, "/")
			updates.Insert(model.ConfigKey{Kind: kind.ServiceEntry, Name: hostname, Namespace: ns})
		}), false)
}

func PushAllXds[T any](xds model.XDSUpdater, f func(T, sets.Set[model.ConfigKey])) func(events []krt.Event[T], initialSync bool) {
	return func(events []krt.Event[T], initialSync bool) {
		cu := sets.New[model.ConfigKey]()
		for _, e := range events {
			for _, i := range e.Items() {
				f(i, cu)
			}
		}
		if len(cu) == 0 {
			return
		}
		xds.ConfigUpdate(&model.PushRequest{
			Full:           false,
			ConfigsUpdated: cu,
			Reason:         model.NewReasonStats(model.AmbientUpdate),
		})
	}
}

func (a *index) ServicesWithWaypoint(key string) []model.ServiceWaypointInfo {
	res := []model.ServiceWaypointInfo{}
	var svcs []model.ServiceInfo
	if key == "" {
		svcs = a.services.List()
	} else {
		svcs = ptr.ToList(a.services.GetKey(krt.Key[model.ServiceInfo](key)))
	}
	for _, s := range svcs {
		wp := s.GetWaypoint()
		if wp == nil {
			continue
		}
		wpAddr, _ := netip.AddrFromSlice(wp.GetAddress().Address)
		wpNetAddr := networkAddress{
			network: wp.GetAddress().Network,
			ip:      wpAddr.String(),
		}
		waypoints := a.services.ByAddress.Lookup(wpNetAddr)
		if len(waypoints) == 0 {
			// No waypoint found. TODO: support passing WaypointAddress
			continue
		}
		wi := model.ServiceWaypointInfo{
			Service:          s.Service,
			WaypointHostname: waypoints[0].Hostname,
		}
		res = append(res, wi)
	}
	return res
}
