//go:build integ
// +build integ

// Copyright Solo.io, Inc
//
// Licensed under a Solo commercial license, not Apache License, Version 2 or any other variant

package ambient

import (
	"context"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"istio.io/istio/pkg/config/protocol"
	"istio.io/istio/pkg/test/echo/common/scheme"
	"istio.io/istio/pkg/test/framework"
	"istio.io/istio/pkg/test/framework/components/echo"
	"istio.io/istio/pkg/test/framework/components/echo/check"
	"istio.io/istio/pkg/test/framework/components/echo/common/ports"
	"istio.io/istio/pkg/test/framework/components/istio"
	"istio.io/istio/pkg/test/scopes"
)

func TestInterop(t *testing.T) {
	framework.NewTest(t).Run(func(t framework.TestContext) {
		// Apply a deny-all waypoint policy. This allows us to test the traffic traverses the waypoint
		t.ConfigIstio().Eval(apps.Namespace.Name(), map[string]string{
			"Waypoint": apps.ServiceAddressedWaypoint.Config().ServiceWaypointProxy,
		}, `
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: deny-all-waypoint
spec:
  targetRefs:
  - kind: Gateway
    group: gateway.networking.k8s.io
    name: {{.Waypoint}}
`).ApplyOrFail(t)
		t.NewSubTest("sidecar-service").Run(func(t framework.TestContext) {
			for _, src := range apps.Sidecar {
				for _, dst := range apps.ServiceAddressedWaypoint {
					for _, opt := range callOptions {
						t.NewSubTestf("%v", opt.Scheme).Run(func(t framework.TestContext) {
							opt = opt.DeepCopy()
							opt.To = dst
							opt.Check = CheckDeny
							src.CallOrFail(t, opt)
						})
					}
				}
			}
		})
		t.NewSubTest("sidecar-workload").Run(func(t framework.TestContext) {
			t.Skip("not yet implemented")
			for _, src := range apps.Sidecar {
				for _, dst := range apps.WorkloadAddressedWaypoint {
					for _, dstWl := range dst.WorkloadsOrFail(t) {
						for _, opt := range callOptions {
							t.NewSubTestf("%v-%v", opt.Scheme, dstWl.Address()).Run(func(t framework.TestContext) {
								opt = opt.DeepCopy()
								opt.Address = dstWl.Address()
								opt.Port = echo.Port{ServicePort: ports.All().MustForName(opt.Port.Name).WorkloadPort}
								opt.Check = CheckDeny
								src.CallOrFail(t, opt)
							})
						}
					}
				}
			}
		})
		t.NewSubTest("ingress-service").Run(func(t framework.TestContext) {
			t.ConfigIstio().Eval(apps.Namespace.Name(), map[string]string{
				"Destination": apps.ServiceAddressedWaypoint.ServiceName(),
			}, `apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: gateway
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts: ["*"]
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: route
spec:
  gateways:
  - gateway
  hosts:
  - "*"
  http:
  - route:
    - destination:
        host: "{{.Destination}}"
`).ApplyOrFail(t)
			ingress := istio.DefaultIngressOrFail(t, t)
			t.NewSubTest("endpoint routing").Run(func(t framework.TestContext) {
				ingress.CallOrFail(t, echo.CallOptions{
					Port: echo.Port{
						Protocol:    protocol.HTTP,
						ServicePort: 80,
					},
					Scheme: scheme.HTTP,
					Check:  check.OK(),
				})
			})
			t.NewSubTest("service routing").Run(func(t framework.TestContext) {
				SetServiceAddressed(t, apps.ServiceAddressedWaypoint.ServiceName(), apps.ServiceAddressedWaypoint.NamespaceName())
				ingress.CallOrFail(t, echo.CallOptions{
					Port: echo.Port{
						Protocol:    protocol.HTTP,
						ServicePort: 80,
					},
					Scheme: scheme.HTTP,
					Check:  CheckDeny,
				})
			})
		})
		t.NewSubTest("ingress-workload").Run(func(t framework.TestContext) {
			t.Skip("not implemented")
			t.ConfigIstio().Eval(apps.Namespace.Name(), map[string]string{
				"Destination": apps.WorkloadAddressedWaypoint.ServiceName(),
			}, `apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: gateway
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts: ["*"]
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: route
spec:
  gateways:
  - gateway
  hosts:
  - "*"
  http:
  - route:
    - destination:
        host: "{{.Destination}}"
`).ApplyOrFail(t)
			ingress := istio.DefaultIngressOrFail(t, t)
			t.NewSubTest("endpoint routing").Run(func(t framework.TestContext) {
				ingress.CallOrFail(t, echo.CallOptions{
					Port: echo.Port{
						Protocol:    protocol.HTTP,
						ServicePort: 80,
					},
					Scheme: scheme.HTTP,
					Check:  CheckDeny,
				})
			})
			t.NewSubTest("service routing").Run(func(t framework.TestContext) {
				// This will be ignored entirely if there is only workload waypoint, so this behaves the same as endpoint routing.
				SetServiceAddressed(t, apps.WorkloadAddressedWaypoint.ServiceName(), apps.WorkloadAddressedWaypoint.NamespaceName())
				ingress.CallOrFail(t, echo.CallOptions{
					Port: echo.Port{
						Protocol:    protocol.HTTP,
						ServicePort: 80,
					},
					Scheme: scheme.HTTP,
					Check:  CheckDeny,
				})
			})
		})
	})
}

func SetServiceAddressed(t framework.TestContext, name, ns string) {
	for _, c := range t.Clusters() {
		set := func(service bool) error {
			var set string
			if service {
				set = fmt.Sprintf("%q", "true")
			} else {
				set = "null"
			}
			label := []byte(fmt.Sprintf(`{"metadata":{"labels":{"%s":%s}}}`,
				"istio.io/ingress-use-waypoint", set))
			_, err := c.Kube().CoreV1().Services(ns).Patch(context.TODO(), name, types.MergePatchType, label, metav1.PatchOptions{})
			return err
		}

		if err := set(true); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := set(false); err != nil {
				scopes.Framework.Errorf("failed resetting service-addressed for %s", name)
			}
		})
	}
}
