package controllers

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"os"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func TestWavefrontOperatorReconciler_provisionProxy(t *testing.T) {
	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
	}
	type args struct {
		ctx context.Context
		req controllerruntime.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "provision fails if",
			fields:  fields{},
			args:    args{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &WavefrontOperatorReconciler{
				Client: tt.fields.Client,
				Scheme: tt.fields.Scheme,
			}
			if err := r.provisionProxy(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("provisionProxy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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
			got, err := KubernetesFilePaths(tt.dir)
			assert.Equal(t, tt.want, len(got))
			if err != nil {
				assert.Contains(t, err.Error(), tt.err.Error())
			}
		})
	}
}
