package controllers_test

import (
	"errors"
	"github.com/stretchr/testify/assert"
	wavefrontcomv1 "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/controllers"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"testing/fstest"
	"time"
)

func TestKubernetesFilePaths(t *testing.T) {
	emptyDir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	fileDir, err := ioutil.TempDir("", "temp")
	if err != nil {
		log.Fatal(err)
	}
	ioutil.TempFile(fileDir, "*.yaml")
	ioutil.TempFile(fileDir, "*.txt")

	defer os.RemoveAll(emptyDir)
	defer os.RemoveAll(fileDir)

	tests := []struct {
		name string
		dir  string
		want int
		err  error
	}{
		{"Invalid directory", "/invalidDir", 0, errors.New("no such file or directory")},
		{"Empty directory", emptyDir, 0, nil},
		{"Directory with txt and yaml files", fileDir, 1, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := controllers.ResourceFiles(tt.dir)
			assert.Equal(t, tt.want, len(got))
			if err != nil {
				assert.Contains(t, err.Error(), tt.err.Error())
			}
		})
	}
}

func TestReadAndInterpolateResources(t *testing.T) {
	t.Run("Interpolate multiple files", func(t *testing.T) {
		spec := wavefrontcomv1.WavefrontOperatorSpec{
			ClusterName:    "fake-cluster-name",
			WavefrontToken: "fake-token",
		}
		fakeFiles := fstest.MapFS{
			"proxy.yaml": &fstest.MapFile{
				Data:    []byte("whatIsNameProxy: {{.ClusterName}}"),
				Mode:    fs.ModePerm,
				ModTime: time.Now(),
				Sys:     nil,
			},
			"config-map.yaml": &fstest.MapFile{
				Data:    []byte("whatIsNameConfig: {{.ClusterName}}"),
				Mode:    fs.ModePerm,
				ModTime: time.Now(),
				Sys:     nil,
			},
			"collector.yaml": &fstest.MapFile{
				Data:    []byte("whatIsNameCollector: {{.ClusterName}}"),
				Mode:    fs.ModePerm,
				ModTime: time.Now(),
				Sys:     nil,
			},
		}
		resources, _ := controllers.ReadAndInterpolateResources(fakeFiles, spec, []string{"proxy.yaml", "config-map.yaml", "collector.yaml"})
		assert.Equal(t, resources[0], "whatIsNameProxy: fake-cluster-name")
		assert.Equal(t, resources[1], "whatIsNameConfig: fake-cluster-name")
		assert.Equal(t, resources[2], "whatIsNameCollector: fake-cluster-name")
	})

	t.Run("Handles non-parsable templates", func(t *testing.T) {
		spec := wavefrontcomv1.WavefrontOperatorSpec{
			ClusterName:    "fake-cluster-name",
			WavefrontToken: "fake-token",
		}
		emptyFS := fstest.MapFS{}
		_, err := controllers.ReadAndInterpolateResources(emptyFS, spec, []string{"some.yaml"})
		assert.Error(t, err, "Expected template error")
	})

	t.Run("Handles non-executable templates", func(t *testing.T) {
		spec := wavefrontcomv1.WavefrontOperatorSpec{
			ClusterName:    "fake-cluster-name",
			WavefrontToken: "fake-token",
		}
		fakeFiles := fstest.MapFS{
			"some.yaml": &fstest.MapFile{
				Data:    []byte("someKey: {{.NonExistentField}}"),
				Mode:    fs.ModePerm,
				ModTime: time.Now(),
				Sys:     nil,
			},
		}
		_, err := controllers.ReadAndInterpolateResources(fakeFiles, spec, []string{"some.yaml"})
		assert.Error(t, err, "Expected execution error")
	})
}
