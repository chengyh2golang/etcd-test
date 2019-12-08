package service

import (
	"etcd-test/pkg/apis/app/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)



func New(etcd *v1alpha1.Etcd) *corev1.Service {
	return &corev1.Service{}
}

