package wavefrontupgradeutil

import (
	"net/http"
	"github.com/tidwall/gjson"
	"fmt"
	"io/ioutil"
	"strings"
	"strconv"
	"github.com/prometheus/common/log"
)

const (
	ProxyImageName = "proxy"

	CollectorImageName = "wavefront-kubernetes-collector"
)

func GetLatestVersion(crImageName string, currentVersion string) (string, error) {
	if currentVersion == "latest" {
		return "", nil
	}

	url := "https://registry.hub.docker.com/v2/repositories/wavefronthq/" + crImageName + "/tags/"
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
	// Ex: [latest 5.5 5.1 4.38 4.36 4.35 4.34 4.33 4.32 4.31]
	versions := gjson.Get(string(contents), "results.#.name")


	majorVersion := strings.Split(currentVersion, ".")[0]
	currVer, err := strconv.ParseFloat(currentVersion, 64)
	if err != nil {
		return "", err
	}

	// Filter based on major version and then minor version (Also, should be non-"rc" build).
	var latestV gjson.Result
	latestminorversion := ""
	foundUpgradeVersion :=  false
	for _, v := range versions.Array() {
		if strings.HasPrefix(v.String(), majorVersion) && !strings.Contains(v.String(), "rc") {
			if currVer < v.Float() {
				currVer = v.Float()
				latestV = v
				foundUpgradeVersion = true
			}
		}
	}

	if foundUpgradeVersion {
		latestminorversion = latestV.Str
		log.Info("Found new Minor Upgrade version :: " + latestminorversion + ", current version " + currentVersion)
	}

	//if strings.Compare(latestminorversion, currentVersion) == 0 {
	//
	//}

	return latestminorversion, nil
}