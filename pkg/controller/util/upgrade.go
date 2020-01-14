// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/go-logr/logr"
	"github.com/tidwall/gjson"
)

const (
	ProxyImageName = "proxy"

	CollectorImageName = "wavefront-kubernetes-collector"

	DockerHubImagePrefix = "wavefronthq/"

	TestSuffix = "-test"
)

// GetLatestVersion checks for auto upgrade eligibility and returns the latest minor version as applicable.
func GetLatestVersion(crImage string, reqLogger logr.Logger) (string, error) {
	imgSlice := strings.Split(crImage, ":")
	crImageName := imgSlice[0]
	currentVersion := imgSlice[1]

	// Auto upgrade is supported only for official wavefronthq docker hub images.
	// Relax docker hub official repository constraint for image names with suffix "-test".
	if !strings.HasSuffix(crImageName, TestSuffix) {
		if !strings.HasPrefix(crImage, DockerHubImagePrefix) {
			reqLogger.Info("Auto Upgrade not supported,", "Cause :: Not an offically supported wavefronthq Docker Hub Image.", crImage)
			return "", nil
		}
	}

	// "latest" effectively renders auto upgrade useless.
	if currentVersion == "latest" {
		reqLogger.Info("Auto Upgrade not supported,", " Cause :: currentVersion is ", currentVersion)
		return currentVersion, nil
	}

	// The last 20 tags should be good. Don't expect customers to be using a really old version of CR.
	url := "https://registry.hub.docker.com/v2/repositories/" + crImageName + "/tags/?page_size=20"
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected http GET status: %s", resp.Status)
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// The below will get us the versions from json.
	// Ex for Proxy: [latest 5.5 5.1 4.38 4.36 4.35 4.34 4.33 4.32 4.31]
	versions := gjson.Get(string(contents), "results.#.name")

	majorVersion := strings.Split(currentVersion, ".")[0]

	finalSemV, err := semver.NewVersion(currentVersion)
	if err != nil {
		return "", err
	}

	// Filter based on major version and then minor version (Also, should be non-"rc" build).
	foundUpgradeVersion := false
	for _, v := range versions.Array() {
		if strings.HasPrefix(v.String(), majorVersion) && !strings.Contains(v.String(), "rc") && !strings.Contains(v.String(), "beta") {
			if semV, err := semver.NewVersion(v.String()); err == nil {
				if semV.GreaterThan(finalSemV) {
					finalSemV = semV
					foundUpgradeVersion = true
				}
			}
		}
	}

	if foundUpgradeVersion {
		reqLogger.Info("Found newer Minor Upgrade version :: " + finalSemV.Original() + ", " +
			"current version " + currentVersion)
	}

	return finalSemV.Original(), nil
}
