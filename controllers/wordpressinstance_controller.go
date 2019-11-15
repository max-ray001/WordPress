/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	wordpressv1alpha1 "github.com/crossplaneio/sample-stack-wordpress/api/v1alpha1"
)

const (
	longWait  = 1 * time.Minute
	shortWait = 30 * time.Second

	defaultWordpressImage = "wordpress:4.6.1-apache"
)

// WordpressInstanceReconciler reconciles a WordpressInstance object
type WordpressInstanceReconciler struct {
	client.Client
	Log logr.Logger
}

func (r *WordpressInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wordpressv1alpha1.WordpressInstance{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=wordpress.samples.stacks.crossplane.io,resources=wordpressinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wordpress.samples.stacks.crossplane.io,resources=wordpressinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=compute.crossplane.io,resources=kubernetesclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.crossplane.io,resources=mysqlinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workload.crossplane.io,resources=kubernetesapplication,verbs=get;list;watch;create;update;patch;delete

func (r *WordpressInstanceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	wp := &wordpressv1alpha1.WordpressInstance{}
	if err := r.Client.Get(ctx, req.NamespacedName, wp); err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{Requeue: false}, nil
		}
		return ctrl.Result{RequeueAfter: shortWait}, err
	}

	FillWithDefaults(wp)
	if err := ApplyResources(ctx, r.Client, *wp); err != nil {
		return ctrl.Result{RequeueAfter: shortWait}, err
	}
	endpoint, err := GetEndpoint(ctx, r.Client, *wp)
	if err != nil {
		return ctrl.Result{RequeueAfter: shortWait}, err
	}
	if endpoint != "" {
		wp.Status.Endpoint = endpoint
	}
	if err := r.Client.Status().Update(ctx, wp); err != nil {
		return ctrl.Result{RequeueAfter: shortWait}, err
	}
	return ctrl.Result{RequeueAfter: longWait}, nil
}

func ApplyResources(ctx context.Context, kube client.Client, wp wordpressv1alpha1.WordpressInstance) error {
	database := ProduceMySQLInstance(wp)
	if err := kube.Patch(ctx, &database, client.Apply); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot create %s/%s in namespace %s", database.Kind, database.GetName(), database.GetNamespace()))
	}
	cluster := ProduceKubernetesCluster(wp)
	if err := kube.Patch(ctx, &cluster, client.Apply); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot create %s/%s in namespace %s", cluster.Kind, cluster.GetName(), cluster.GetNamespace()))
	}
	kubernetesApplication := ProduceKubernetesApplication(wp, database.GetName())
	if err := kube.Patch(ctx, &kubernetesApplication, client.Apply); err != nil {
		return errors.Wrap(err, fmt.Sprintf("cannot create %s/%s in namespace %s", kubernetesApplication.Kind, kubernetesApplication.GetName(), kubernetesApplication.GetNamespace()))
	}
	return nil
}

func GetEndpoint(ctx context.Context, kube client.Client, wp wordpressv1alpha1.WordpressInstance) (string, error) {
	serviceResource := &v1.Service{}
	key, err := GetServiceResourceObjectKey(wp)
	if err != nil {
		return "", err
	}
	if err := kube.Get(ctx, key, serviceResource); err != nil {
		return "", err
	}
	ingress := serviceResource.Status.LoadBalancer.Ingress[0]
	if ingress.IP != "" {
		return ingress.IP, nil
	}
	if ingress.Hostname != "" {
		return ingress.Hostname, nil
	}
	return "", nil
}

func GetServiceResourceObjectKey(wp wordpressv1alpha1.WordpressInstance) (client.ObjectKey, error) {
	kubernetesApplication := ProduceKubernetesApplication(wp, "")
	for _, template := range kubernetesApplication.Spec.ResourceTemplates {
		if template.Spec.Template.GetKind() == "Service" {
			return client.ObjectKey{Name: template.Spec.Template.GetName(), Namespace: template.Spec.Template.GetNamespace()}, nil
		}
	}
	return client.ObjectKey{}, fmt.Errorf("cannot find service type kubernetes application resource for %s/%s in namespace %s", wp.Kind, wp.GetName(), wp.GetNamespace())
}

func FillWithDefaults(wp *wordpressv1alpha1.WordpressInstance) {
	if wp.Spec.Image == "" {
		wp.Spec.Image = defaultWordpressImage
	}
}

// ConvertToUnstructured takes a metav1.Object and produces a *unstructured.Unstructured
// object.
func ConvertToUnstructured(o metav1.Object) *unstructured.Unstructured {
	// NOTE(muvaf): Only constraint of unstructured.Unstructured object is that
	// the object should have metadata and using metav1.Object ensures that. So,
	// it should be safe to cast.
	u, _ := o.(*unstructured.Unstructured)
	return u
}
