package hook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	v2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	clientset "github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	v1 "github.com/cilium/cilium/pkg/k8s/slim/k8s/apis/meta/v1"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/rueian/aerial/pkg/tunnel"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"net"
	"regexp"
)

const CiliumNamespace = "default"

func OnBind(msg tunnel.Message, addr net.Addr) (interface{}, error) {
	init := map[string]map[string]string{}
	if err := json.Unmarshal(msg.Body, &init); err != nil {
		return nil, err
	}

	if v, ok := init["params"]; !ok || len(v) == 0 {
		return nil, errors.New("init params is required")
	}

	if v, ok := init["labels"]; !ok || len(v) == 0 {
		return nil, errors.New("init labels is required")
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				init["params"]["ProxyAddr"] = fmt.Sprintf("%s:%d", ipnet.IP.String(), addr.(*net.TCPAddr).Port)
				goto crd
			}
		}
	}
	return nil, errors.New("ip not found")
crd:

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	client, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	var name string
	for k, v := range init["labels"] {
		name += k + "-" + v + "-"
	}
	name += regexp.MustCompile("[:.\\[\\]]").ReplaceAllString(init["params"]["ProxyAddr"], "-")

	return client.CiliumV2().CiliumNetworkPolicies(CiliumNamespace).Create(context.Background(), &v2.CiliumNetworkPolicy{
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
				LabelSelector: &v1.LabelSelector{MatchLabels: init["labels"]},
			},
			Ingress: []api.IngressRule{{
				FromEndpoints: []api.EndpointSelector{{LabelSelector: &v1.LabelSelector{}}},
				ToPorts: []api.PortRule{{
					Ports: []api.PortProtocol{},
					Rules: &api.L7Rules{
						L7Proto: "HTTPRedirect",
						L7:      []api.PortRuleL7{init["params"]},
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
	client, err := clientset.NewForConfig(config)
	if err != nil {
		return err
	}

	return client.CiliumV2().CiliumNetworkPolicies(CiliumNamespace).
		Delete(context.Background(), cnp.Name, metav1.DeleteOptions{})
}
