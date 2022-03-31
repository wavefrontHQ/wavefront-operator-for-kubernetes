package controllers

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
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
	},
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