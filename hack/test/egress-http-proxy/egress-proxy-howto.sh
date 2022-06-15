# create the wavefront
kubectl create namespace wavefront

# deploy the mitmproxy
kubectl apply -f ~/workspace/wavefront-operator-for-kubernetes/hack/test/egress-http-proxy/egress-proxy.yaml

# wait for egress proxy
sleep 60

# port forward to the proxy from localhost
kubectl -n wavefront port-forward svc/egress-proxy 8080:8080

# get the ca cert efor the mitmpproxy
export MITM_POD="$(kubectl -n wavefront get pods --selector=app=egress-proxy -o jsonpath='{.items[*].metadata.name}')"
kubectl cp wavefront/${MITM_POD}:root/.mitmproxy/mitmproxy-ca-cert.pem ~/workspace/wavefront-operator-for-kubernetes/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem

# connect to the proxy with TLS and MITM
curl -LI -vvv --proxy https://localhost:8080 --proxy-cacert ~/workspace/wavefront-operator-for-kubernetes/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem --cacert ~/workspace/wavefront-operator-for-kubernetes/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem https://www.google.com/ &> tls-mitm-curl-output.txt

# connect to the proxy without TLS and MITM
curl -LI -vvv --proxy http://localhost:8080 --cacert ~/workspace/wavefront-operator-for-kubernetes/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem https://www.google.com/ &> without-tls-with-mitm-curl-output.txt