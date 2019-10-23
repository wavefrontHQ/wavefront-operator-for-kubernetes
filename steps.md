The operator was created following the steps documented under the operator-sdk [user-guide](https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md).

The steps specific to the wavefront operator are captured here:

1. Install the operator-sdk

2. Create a new wavefront-operator project:
```
$ cd  $HOME/dev
$ operator-sdk new wavefront-operator --repo=github.com/wavefronthq/wavefront-operator
$ cd wavefront-operator
```

3. Add a new CRD for the collector:
```
$ operator-sdk add api --api-version=wavefront.com/v1alpha1 --kind=WavefrontCollector
```

4. Manually modify the spec and status at `/pkg/apis/wavefront/v1alpha1/wavefrontcollector_types.go`
Run `operator-sdk generate k8s` after modifying the spec and status.

5. Run `operator-sdk generate openapi` to update the OpenAPIValidation section in the CRD. This updates the CRD based on latest spec / status mentioned in `/pkg/apis/wavefront/v1alpha1/wavefrontcollector_types.go`

6. Add a new controller for the collector
```
operator-sdk add controller --api-version=wavefront.com/v1alpha1 --kind=WavefrontCollector
```
Flesh out the controller logic as relevant to the CRD.

7. Build the docker image:
```
operator-sdk build ${REPO_NAME}/wavefront-operator:latest
```

8. Deploy the operator:
```
$ ka -f deploy/crds/wavefront_v1alpha1_wavefrontcollector_crd.yaml
$ ka -f deploy/service_account.yaml
$ ka -f deploy/role.yaml
$ ka -f deploy/role_binding.yaml
$ ka -f deploy/operator.yaml
```

9. Now deploying a wavefrontcollector CR should create pods for the collector:
```
ka -f deploy/crds/wavefront_v1alpha1_wavefrontcollector_cr.yaml
```
