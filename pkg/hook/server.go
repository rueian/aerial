package hook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	v2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	cilium "github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	v1 "github.com/cilium/cilium/pkg/k8s/slim/k8s/apis/meta/v1"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/rueian/aerial/pkg/tunnel"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"net"
	"regexp"
	"strconv"
)

const CiliumNamespace = "default"

type Init struct {
	Svc    string
	Params map[string]string
}

func OnBind(msg tunnel.Message, addr net.Addr) (interface{}, error) {
	init := Init{}
	if err := json.Unmarshal(msg.Body, &init); err != nil {
		return nil, err
	}

	if len(init.Svc) == 0 {
		return nil, errors.New("init svc is required")
	}

	if len(init.Params) == 0 {
		return nil, errors.New("init params is required")
	}

	host, port, err := net.SplitHostPort(init.Svc)
	if err != nil {
		return nil, err
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				init.Params["ProxyAddr"] = fmt.Sprintf("%s:%d", ipnet.IP.String(), addr.(*net.TCPAddr).Port)
				goto port
			}
		}
	}
	return nil, errors.New("ip not found")
port:

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	ciliumClient, err := cilium.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	coreClient, err := corev1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	svc, err := coreClient.Services(CiliumNamespace).Get(ctx, host, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var targetPort apiv1.ServicePort
	for _, p := range svc.Spec.Ports {
		if ps := strconv.Itoa(int(p.Port)); port == ps {
			targetPort = p
			goto crd
		}
	}
	return nil, errors.New("service port not found")
crd:

	name := regexp.MustCompile("[:.\\[\\]]").ReplaceAllString(init.Svc+"-"+init.Params["ProxyAddr"], "-")

	return ciliumClient.CiliumV2().CiliumNetworkPolicies(CiliumNamespace).Create(ctx, &v2.CiliumNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CiliumNetworkPolicy",
			APIVersion: "cilium.io/v2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: CiliumNamespace,
		},
		Spec: &api.Rule{
			EndpointSelector: api.EndpointSelector{
				LabelSelector: &v1.LabelSelector{MatchLabels: svc.Spec.Selector},
			},
			Ingress: []api.IngressRule{{
				FromEndpoints: []api.EndpointSelector{{LabelSelector: &v1.LabelSelector{}}},
				ToPorts: []api.PortRule{{
					Ports: []api.PortProtocol{{
						Port:     targetPort.TargetPort.String(),
						Protocol: api.L4Proto(targetPort.Protocol),
					}},
					Rules: &api.L7Rules{
						L7Proto: "HTTPRedirect",
						L7:      []api.PortRuleL7{init.Params},
					},
				}},
			}},
			Labels: nil,
		},
	}, metav1.CreateOptions{})
}

func OnClose(in interface{}) error {
	cnp, ok := in.(*v2.CiliumNetworkPolicy)
	if !ok {
		return nil
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	client, err := cilium.NewForConfig(config)
	if err != nil {
		return err
	}

	return client.CiliumV2().CiliumNetworkPolicies(CiliumNamespace).
		Delete(context.Background(), cnp.Name, metav1.DeleteOptions{})
}
