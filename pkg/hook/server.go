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
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"log"
	"net"
	"regexp"
	"strconv"
	"time"
)

const CiliumNamespace = "default"

type Init struct {
	Svc    string
	Params map[string]string
}

func OnBind(msg tunnel.Message, addr net.Addr) (res interface{}, err error) {
	init := &Init{}
	if err := json.Unmarshal(msg.Body, init); err != nil {
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

	policy := &v2.CiliumNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CiliumNetworkPolicy",
			APIVersion: "cilium.io/v2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      regexp.MustCompile("[:.\\[\\]]").ReplaceAllString("aerial-tunnel-"+init.Svc, "-"),
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
		},
	}
	for i := 0; i == 0 || err != nil; i++ {
		if i != 0 {
			log.Println(err)
			time.Sleep(time.Second)
		}
		res, err := ciliumClient.CiliumV2().CiliumNetworkPolicies(CiliumNamespace).Get(ctx, policy.Name, metav1.GetOptions{})
		if kerrs.IsNotFound(err) {
			res, err = ciliumClient.CiliumV2().CiliumNetworkPolicies(CiliumNamespace).Create(ctx, policy, metav1.CreateOptions{})
			continue
		}
		if len(res.Spec.Ingress) != 1 || len(res.Spec.Ingress[0].ToPorts) != 1 {
			res.Spec.Ingress = policy.Spec.Ingress
		} else {
			res.Spec.Ingress[0].ToPorts[0].Rules.L7 = append(res.Spec.Ingress[0].ToPorts[0].Rules.L7, init.Params)
		}
		res, err = ciliumClient.CiliumV2().CiliumNetworkPolicies(CiliumNamespace).Update(ctx, res, metav1.UpdateOptions{})
	}
	return init, nil
}

func OnClose(in interface{}) (err error) {
	init, ok := in.(*Init)
	if !ok {
		return nil
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	ciliumClient, err := cilium.NewForConfig(config)
	if err != nil {
		return err
	}

	ctx := context.Background()
	name := regexp.MustCompile("[:.\\[\\]]").ReplaceAllString("aerial-tunnel-"+init.Svc, "-")
	for i := 0; i == 0 || err != nil; i++ {
		if i != 0 {
			log.Println(err)
			time.Sleep(time.Second)
		}
		res, err := ciliumClient.CiliumV2().CiliumNetworkPolicies(CiliumNamespace).Get(ctx, name, metav1.GetOptions{})
		if kerrs.IsNotFound(err) {
			return nil
		}
		var others []api.PortRuleL7
		if len(res.Spec.Ingress) == 1 && len(res.Spec.Ingress[0].ToPorts) == 1 {
			for _, rule := range res.Spec.Ingress[0].ToPorts[0].Rules.L7 {
				if !rule.Equal(init.Params) {
					others = append(others, rule)
				}
			}
		}
		if len(others) == 0 {
			err = ciliumClient.CiliumV2().CiliumNetworkPolicies(CiliumNamespace).Delete(ctx, name, metav1.DeleteOptions{})
		} else {
			res.Spec.Ingress[0].ToPorts[0].Rules.L7 = others
			res, err = ciliumClient.CiliumV2().CiliumNetworkPolicies(CiliumNamespace).Update(ctx, res, metav1.UpdateOptions{})
		}
	}
	return nil
}
