#!/usr/bin/env bash

go run hack/test/validation/main.go <(yq .spec.versions.0.schema.openAPIV3Schema config/crd/bases/wavefront.com_wavefronts.yaml)
