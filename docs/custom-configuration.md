# Deploy the Wavefront Operator with a custom registry

Install the Wavefront Operator into `observability-system` namespace.

**Note**: All the components use the same image registry in the operator. Copy over the below component images to your image registry.
- Wavefront kubernetes operator:`projects.registry.vmware.com/tanzu_observability/kubernetes-operator:2.0.0-rc01` to `YOUR_IMAGE_REGISTRY/kubernetes-operator:2.0.0-rc01`
- Wavefront kubernetes collector: `projects.registry.vmware.com/tanzu_observability/kubernetes-collector:1.12.0` to `YOUR_IMAGE_REGISTRY/kubernetes-collector:1.12.0`
- Wavefront Proxy: `projects.registry.vmware.com/tanzu_observability/proxy:12.0` to `YOUR_IMAGE_REGISTRY/proxy:12.0`
- Wavefront logging:`projects.registry.vmware.com/tanzu_observability/kubernetes-operator-fluentd:1.0.4-1.15.2` to `YOUR_IMAGE_REGISTRY/kubernetes-operator-fluentd:1.0.4-1.15.2`

1. Create a directory
2. Download [wavefront-operator.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/wavefront-operator.yaml) into the directory created.
3. In the directory create a `kustomization.yaml` file.
  ```yaml
  # Need to change YOUR_IMAGE_REGISTRY
  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization
   
  resources:
  - wavefront-operator.yaml
   
  images:
  - name: projects.registry.vmware.com/tanzu_observability/kubernetes-operator
    newName: YOUR_IMAGE_REGISTRY/kubernetes-operator
  ```
4. Deploy the wavefront operator 
  ```
  kubectl apply -k <DIRECTORY>
  ```
5. Now follow from step 2 in [Deploy the Wavefront Collector and Proxy with the Operator](../README.md#deploy-the-wavefront-collector-and-proxy-with-the-operator)