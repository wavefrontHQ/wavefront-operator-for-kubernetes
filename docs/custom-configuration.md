# Deploy the Wavefront Operator with a custom registry

Install the Wavefront Operator into `observability-system` namespace.

**Note**: All the integration components use the same image registry in the operator.

1. Copy the following images over to `YOUR_IMAGE_REGISTRY`, keeping the same repos and tags.

| Component                      | From                                                                                        | To                                                             |
|--------------------------------|---------------------------------------------------------------------------------------------|----------------------------------------------------------------|
| Wavefront kubernetes operator  | `projects.registry.vmware.com/tanzu_observability/kubernetes-operator:2.0.0-rc01`           | `YOUR_IMAGE_REGISTRY/kubernetes-operator:2.0.0-rc01`           |
| Wavefront kubernetes collector | `projects.registry.vmware.com/tanzu_observability/kubernetes-collector:1.12.0`              | `YOUR_IMAGE_REGISTRY/kubernetes-collector:1.12.0`              |
| Wavefront Proxy                | `projects.registry.vmware.com/tanzu_observability/proxy:12.0`                               | `YOUR_IMAGE_REGISTRY/proxy:12.0`                               |
| Wavefront logging              | `projects.registry.vmware.com/tanzu_observability/kubernetes-operator-fluentd:1.0.4-1.15.2` | `YOUR_IMAGE_REGISTRY/kubernetes-operator-fluentd:1.0.4-1.15.2` |

2. Create a local directory called `observability`
3. Download [wavefront-operator.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/wavefront-operator.yaml) into the `observability` directory.
4. Create a `kustomization.yaml` file in the `observability` directory.
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
5. Deploy the wavefront operator 
  ```
  kubectl apply -k observability
  ```
6. Now follow from step 2 in [Deploy the Wavefront Collector and Proxy with the Operator](../README.md#deploy-the-wavefront-collector-and-proxy-with-the-operator)

# Deploy the Wavefront Operator into a custom namespace

1. Create a local directory called `observability`
2. Download [wavefront-operator.yaml](https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/main/deploy/kubernetes/wavefront-operator.yaml) into the `observability` directory.
3. Create a `kustomization.yaml` file in the `observability` directory.
  ```yaml
  # Need to change YOUR_NAMESPACE
  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization

  resources:
  - wavefront-operator.yaml

  namespace: YOUR_NAMESPACE
  patches:
  - target:
       kind: RoleBinding
    patch: |-
       - op: replace
         path: /subjects/0/namespace
         value: YOUR_NAMESPACE
  - target:
       kind: ClusterRoleBinding
    patch: |-
       - op: replace
         path: /subjects/0/namespace
         value: YOUR_NAMESPACE
  ```
4. Deploy the wavefront operator
  ```
  kubectl apply -k observability
  ```
5. Now follow from step 2 in [Deploy the Wavefront Collector and Proxy with the Operator](../README.md#deploy-the-wavefront-collector-and-proxy-with-the-operator),
   replacing `observability-system` with `YOUR_NAMESPACE`.