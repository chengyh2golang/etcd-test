package statefulset

import (
	"etcd-test/pkg/apis/app/v1alpha1"
	_ "fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

)

func New(etcd *v1alpha1.Etcd) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Statefulset",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      etcd.Name,
			Namespace: etcd.Namespace,
			Labels:    map[string]string{"app.example.com": etcd.Name},
		},
		Spec:appsv1.StatefulSetSpec{
			//这个service是headless的svc
			ServiceName:etcd.Name,
			Replicas:etcd.Spec.Replicas,
			Selector:&metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.example.com": etcd.Name,
				},
			},
			Template:corev1.PodTemplateSpec{
				ObjectMeta:metav1.ObjectMeta{
					Name:etcd.Name,
					Labels: map[string]string{
						"app.example.com/v1alpha1":etcd.Name,
					},

				},
				Spec:corev1.PodSpec{
					Containers:[]corev1.Container{},
				},
			},
			//先注释掉，因为测试使用的是本地存储:emptyDir{}
			/*
			VolumeClaimTemplates:[]corev1.PersistentVolumeClaim{
				{
					ObjectMeta:metav1.ObjectMeta{
						Name:"dataDir",
					},
					Spec:corev1.PersistentVolumeClaimSpec{
						AccessModes:[]corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources:corev1.ResourceRequirements{
							Requests:corev1.ResourceList{
								corev1.ResourceStorage:resource.MustParse(
									fmt.Sprintf("%vGi",etcd.Spec.Storage)),
							},
						},
					},
				},
			},*/
		},
	}
}
