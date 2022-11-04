package util

import "sigs.k8s.io/controller-runtime/pkg/client"

func ObjKey(namespace string, name string) client.ObjectKey {
	return client.ObjectKey{Namespace: namespace, Name: name}
}
