#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source ${REPO_ROOT}/hack/test/k8s-utils.sh

# create the namespace observability-system 
kubectl create namespace observability-system || true

# deploy the mitmproxy
kubectl apply -f ${REPO_ROOT}/hack/test/egress-http-proxy/egress-proxy.yaml

# wait for egress proxy
wait_for_cluster_ready


#get httpproxy ip
export MITM_IP="$(kubectl -n observability-system get services --selector=app=egress-proxy -o jsonpath='{.items[*].spec.clusterIP}')"
green "HTTP Proxy URL:"
echo "http://${MITM_IP}:8080"

# get the ca cert efor the mitmpproxy
export MITM_POD="$(kubectl -n observability-system get pods --selector=app=egress-proxy -o jsonpath='{.items[*].metadata.name}')"
kubectl cp wavefront/${MITM_POD}:root/.mitmproxy/mitmproxy-ca-cert.pem ${REPO_ROOT}/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem

green "HTTP Proxy CAcert:"
cat ${REPO_ROOT}/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem
