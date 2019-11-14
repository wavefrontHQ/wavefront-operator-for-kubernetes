package util

import (
	"testing"
	"github.com/Masterminds/semver"
)

func TestProxyValidUpgrade(t *testing.T) {
	// Proxy
	v := "5.1"
	semV, _ := semver.NewVersion(v)
	returnVer, err := GetLatestVersion(ProxyImageName, v, true)
	if err != nil {
		t.Error("Failed to get latest version :: ", err)
	}
	returnSemV, _ := semver.NewVersion(returnVer)

	if returnSemV.LessThan(semV) || returnSemV.Equal(semV) {
		t.Error("Error :: Expected returned version for Proxy Upgrade : ", returnVer,
			" to be greater than input version : " , v)
	}
}



func TestCollectorValidUpgrade(t *testing.T) {
	// Collector
	v := "1.0.0"
	semV, _ := semver.NewVersion(v)
	returnVer, err := GetLatestVersion(CollectorImageName, v, true)
	if err != nil {
		t.Error("Failed to get latest version :: ", err)
	}
	returnSemV, _ := semver.NewVersion(returnVer)

	if returnSemV.LessThan(semV) || returnSemV.Equal(semV) {
		t.Error("Error :: Expected returned version for Collector Upgrade : ", returnVer,
			" to be greater than input version : " , v)
	}
}

func TestImageLatest(t *testing.T) {
	// Proxy
	v := "latest"
	returnVer, err := GetLatestVersion(ProxyImageName, v, true)
	if err != nil {
		t.Error("Failed to get latest version :: ", err)
	}

	if v != returnVer {
		t.Error("Error :: Expected returned version for Proxy Upgrade : ", returnVer,
			" to be same as input version : " , v)
	}
}

func TestUpgradeDisabled(t *testing.T) {
	// Proxy
	v := "2.1"
	semV, _ := semver.NewVersion(v)
	returnVer, err := GetLatestVersion(ProxyImageName, v, false)
	if err != nil {
		t.Error("Failed to get latest version :: ", err)
	}
	returnSemV, _ := semver.NewVersion(returnVer)

	if !returnSemV.Equal(semV) {
		t.Error("Error :: Expected returned version for Proxy Upgrade : ", returnVer,
			" to be same as input version : " , v)
	}
}

