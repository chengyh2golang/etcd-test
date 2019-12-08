package statefulset

import (
	"etcd-test/pkg/apis/app/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
)

func New(etcd *v1alpha1.Etcd) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{}
}
