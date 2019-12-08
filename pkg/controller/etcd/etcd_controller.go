package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"etcd-test/pkg/resources/service"
	"etcd-test/pkg/resources/statefulset"

	appv1alpha1 "etcd-test/pkg/apis/app/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"k8s.io/client-go/util/retry"
)

var log = logf.Log.WithName("controller_etcd")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Etcd Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileEtcd{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("etcd-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Etcd
	err = c.Watch(&source.Kind{Type: &appv1alpha1.Etcd{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Etcd
	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appv1alpha1.Etcd{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileEtcd implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileEtcd{}

// ReconcileEtcd reconciles a Etcd object
type ReconcileEtcd struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Etcd object and makes changes based on the state read
// and what is in the Etcd.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileEtcd) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Etcd")

	// Fetch the Etcd instance
	etcd := &appv1alpha1.Etcd{}
	err := r.client.Get(context.TODO(), request.NamespacedName, etcd)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	//判断etcd的DeletionTimestamp是否有值，
	// 如果有值，说明要被删除了，就直接返回，走k8s的垃圾回收机制
	if etcd.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	//如果查到了，并且不是被删除，就判断它所关联的资源是否存在
	// Check if this Pod already exists
	found := &appsv1.StatefulSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: etcd.Name, Namespace: etcd.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {

		headlessSvc := service.New(etcd)
		err = r.client.Create(context.TODO(), headlessSvc)
		if err != nil {
			return reconcile.Result{}, err
		}

		sts := statefulset.New(etcd)
		err = r.client.Create(context.TODO(), sts)
		if err != nil {
			//如果创建sts报错，先把之前创建的headlessSvc删除后再返回错误
			go r.client.Delete(context.TODO(), headlessSvc)
			return reconcile.Result{}, err
		}

		//创建完成之后还得去做一次更新
		//把对应的annotation给更新上，因为后面需要用annotation去做判断是否需要去做更新操作
		etcd.Annotations = map[string]string{
			"app.example.com/spec":toString(etcd),
		}
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.client.Update(context.TODO(), etcd)
		})
		if retryErr != nil {
			fmt.Println(retryErr.Error())
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	//etcd.Annotations["app.example.com/spec"]这是老的信息
	//etcd.spec是最新的信息，使用DeepEqual方法比较是否相等
	if !reflect.DeepEqual(etcd.Spec,toSpec(etcd.Annotations["app.example.com/spec"])) {
		//如果不相等，就需要去更新，更新就是重建sts和svc
		//但是通常是不会去更新svc的，所以把service.New注释掉，只需要更新sts
		//headlessSvc := service.New(etcd)
		sts := statefulset.New(etcd)
		found.Spec = sts.Spec
		//然后就去更新，更新要用retry操作去做
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.client.Update(context.TODO(), found)
		})
		if retryErr != nil {
			return reconcile.Result{}, err //如果retry报错，就返回给下一次处理
		}

	}


	return reconcile.Result{}, nil
}

func toString(etcd *appv1alpha1.Etcd) string {
	bytes, _ := json.Marshal(etcd.Spec)
	return  string(bytes)
}

func toSpec(data string) appv1alpha1.EtcdSpec {
	etcdSpec := appv1alpha1.EtcdSpec{}
	_ = json.Unmarshal([]byte(data), &etcdSpec)
	return etcdSpec
}





// newPodForCR returns a busybox pod with the same name/namespace as the cr
/*
func newPodForCR(cr *appv1alpha1.Etcd) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}*/

