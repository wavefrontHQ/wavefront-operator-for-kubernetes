
# port forward to the proxy from localhost
kubectl -n observability-system port-forward svc/egress-proxy 8080:8080


export REPO_ROOT=$(git rev-parse --show-toplevel)

# connect to the proxy with TLS and MITM
curl -LI -vvv --proxy https://localhost:8080 --proxy-cacert ${REPO_ROOT}/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem --cacert ${REPO_ROOT}/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem https://www.google.com/

# connect to the proxy without TLS and MITM
curl -LI -vvv --proxy http://localhost:8080 --cacert ${REPO_ROOT}/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem https://www.google.com/