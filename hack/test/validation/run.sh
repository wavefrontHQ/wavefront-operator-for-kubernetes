#!/usr/bin/env bash

REPO_ROOT=$(git rev-parse --show-toplevel)

yq .spec.versions.0.schema.openAPIV3Schema ${REPO_ROOT}/config/crd/bases/wavefront.com_wavefronts.yaml > com_wavefront_schema_extract.yaml

go run ${REPO_ROOT}/hack/test/validation/main.go com_wavefront_schema_extract.yaml
