#!/bin/sh

export CLUSTER_NAME=aerial-cilium
export CLUSTER_ZONE=asia-east1-b
export CILIUM_NAMESPACE=cilium

gcloud config set project $GCP_PROJECT
gcloud services enable container.googleapis.com

gcloud container clusters create $CLUSTER_NAME --cluster-version 1.17.13-gke.1400 --num-nodes 2 --machine-type n2-highcpu-4 --enable-ip-alias --zone $CLUSTER_ZONE
gcloud container clusters get-credentials $CLUSTER_NAME --zone $CLUSTER_ZONE

NATIVE_CIDR="$(gcloud container clusters describe $CLUSTER_NAME --zone $CLUSTER_ZONE --format 'value(clusterIpv4Cidr)')"

kubectl create namespace $CILIUM_NAMESPACE

helm repo add cilium https://helm.cilium.io/
helm repo update
helm install cilium cilium/cilium --version 1.9.0 \
  --namespace $CILIUM_NAMESPACE \
  --set nodeinit.enabled=true \
  --set nodeinit.reconfigureKubelet=true \
  --set nodeinit.removeCbrBridge=true \
  --set nodeinit.restartPods=true \
  --set cni.binPath=/home/kubernetes/bin \
  --set gke.enabled=true \
  --set bpf.tproxy=true \
  --set prometheus.enabled=true \
  --set operatorPrometheus.enabled=true \
  --set operator.prometheus.enabled=true \
  --set ipam.mode=kubernetes \
  --set image.repository=docker.io/rueian/cilium-aerial \
  --set image.tag=v1.9.0 \
  --set hostServices.enabled=true \
  --set nativeRoutingCIDR="$NATIVE_CIDR" \
  --set hubble.enabled=true \
  --set hubble.listenAddress=":4244" \
  --set hubble.metrics.enabled="{dns,drop,tcp,flow,icmp,http}" \
  --set hubble.relay.enabled=true \
  --set hubble.ui.enabled=true
