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
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimev1alpha1 "github.com/crossplaneio/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplaneio/crossplane-runtime/pkg/meta"
	compute "github.com/crossplaneio/crossplane/apis/compute/v1alpha1"
	"github.com/crossplaneio/crossplane/apis/database/v1alpha1"
	workload "github.com/crossplaneio/crossplane/apis/workload/v1alpha1"

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
	*runtime.Scheme
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
	if err := CreateMySQLInstance(ctx, r.Client, wp); err != nil {
		wp.Status.SetConditions(runtimev1alpha1.ReconcileError(err))
		return ctrl.Result{RequeueAfter: shortWait}, r.Client.Status().Update(ctx, wp)
	}
	if err := CreateKubernetesCluster(ctx, r.Client, wp); err != nil {
		wp.Status.SetConditions(runtimev1alpha1.ReconcileError(err))
		return ctrl.Result{RequeueAfter: shortWait}, r.Client.Status().Update(ctx, wp)
	}
	if err := CreateKubernetesApplication(ctx, r.Client, r.Scheme, wp); err != nil {
		wp.Status.SetConditions(runtimev1alpha1.ReconcileError(err))
		return ctrl.Result{RequeueAfter: shortWait}, r.Client.Status().Update(ctx, wp)
	}
	if err := r.Client.Update(ctx, wp); err != nil {
		wp.Status.SetConditions(runtimev1alpha1.ReconcileError(err))
		return ctrl.Result{RequeueAfter: shortWait}, r.Client.Status().Update(ctx, wp)
	}

	endpoint, err := GetEndpoint(ctx, r.Client, *wp)
	if err != nil {
		wp.Status.SetConditions(runtimev1alpha1.ReconcileError(err))
		return ctrl.Result{RequeueAfter: shortWait}, r.Client.Status().Update(ctx, wp)
	}
	wp.Status.Endpoint = endpoint

	wp.Status.SetConditions(runtimev1alpha1.Creating())
	if wp.Status.Endpoint != "" {
		wp.Status.SetConditions(runtimev1alpha1.Available())
	}

	wp.Status.SetConditions(runtimev1alpha1.ReconcileSuccess())
	return ctrl.Result{RequeueAfter: longWait}, r.Client.Status().Update(ctx, wp)
}

func CreateMySQLInstance(ctx context.Context, kube client.Client, wp *wordpressv1alpha1.WordpressInstance) error {
	if wp.Spec.MySQLInstanceRef != nil {
		if err := kube.Get(ctx, meta.NamespacedNameOf(wp.Spec.MySQLInstanceRef), &v1alpha1.MySQLInstance{}); !kerrors.IsNotFound(err) {
			return err
		}
	}
	db := ProduceMySQLInstance(*wp)
	if err := kube.Create(ctx, db); err != nil && !kerrors.IsAlreadyExists(err) {
		return err
	}
	wp.Spec.MySQLInstanceRef = &v1.ObjectReference{
		Name:      db.GetName(),
		Namespace: db.GetNamespace(),
	}
	return nil
}

func CreateKubernetesCluster(ctx context.Context, kube client.Client, wp *wordpressv1alpha1.WordpressInstance) error {
	k8s := &compute.KubernetesCluster{}
	if wp.Spec.KubernetesClusterRef != nil {
		if err := kube.Get(ctx, meta.NamespacedNameOf(wp.Spec.KubernetesClusterRef), k8s); err != nil {
			return err
		}
		// We need to make sure the referenced cluster is picked up by our
		// KubernetesApplication
		meta.AddLabels(k8s, GetLocalResourceSelector(*wp))
		return kube.Update(ctx, k8s)
	}
	k8s = ProduceKubernetesCluster(*wp)
	if err := kube.Create(ctx, k8s); err != nil && !kerrors.IsAlreadyExists(err) {
		return err
	}
	wp.Spec.KubernetesClusterRef = &v1.ObjectReference{
		Name:      k8s.GetName(),
		Namespace: k8s.GetNamespace(),
	}
	return nil
}

func CreateKubernetesApplication(ctx context.Context, kube client.Client, scheme *runtime.Scheme, wp *wordpressv1alpha1.WordpressInstance) error {
	if wp.Spec.KubernetesApplicationRef != nil {
		if err := kube.Get(ctx, meta.NamespacedNameOf(wp.Spec.KubernetesApplicationRef), &workload.KubernetesApplication{}); !kerrors.IsNotFound(err) {
			return err
		}
	}
	if wp.Spec.MySQLInstanceRef == nil {
		return fmt.Errorf("cannot create KubernetesApplication resource without MySQLInstanceRef on WordpressInstance")
	}
	mysql := &v1alpha1.MySQLInstance{}
	if err := kube.Get(ctx, meta.NamespacedNameOf(wp.Spec.MySQLInstanceRef), mysql); err != nil {
		return err
	}
	app, err := ProduceKubernetesApplication(scheme, *wp, mysql.Spec.WriteConnectionSecretToReference)
	if err != nil {
		return err
	}
	if err := kube.Create(ctx, app); err != nil && !kerrors.IsAlreadyExists(err) {
		return err
	}
	wp.Spec.KubernetesApplicationRef = &v1.ObjectReference{
		Name:      app.GetName(),
		Namespace: app.GetNamespace(),
	}
	return nil
}

// GetEndpoint fetches the endpoint of the service that Wordpress pods use.
func GetEndpoint(ctx context.Context, kube client.Client, wp wordpressv1alpha1.WordpressInstance) (string, error) {
	if wp.Spec.KubernetesApplicationRef == nil {
		return "", fmt.Errorf("cannot get the endpoint when there is no KubernetesApplication reference on WordpressInstance")
	}
	app := &workload.KubernetesApplication{}
	if err := kube.Get(ctx, meta.NamespacedNameOf(wp.Spec.KubernetesApplicationRef), app); err != nil {
		return "", err
	}
	serviceResource := &workload.KubernetesApplicationResource{}
	key, err := GetServiceResourceObjectKey(app)
	if err != nil {
		return "", err
	}
	if err := kube.Get(ctx, key, serviceResource); err != nil {
		return "", err
	}
	if serviceResource.Status.Remote == nil {
		return "", nil
	}
	statusInRemote := &v1.ServiceStatus{}
	if err := json.Unmarshal(serviceResource.Status.Remote.Raw, statusInRemote); err != nil {
		return "", err
	}
	if len(statusInRemote.LoadBalancer.Ingress) == 0 {
		return "", nil
	}
	ingress := statusInRemote.LoadBalancer.Ingress[0]
	if ingress.IP != "" {
		return ingress.IP, nil
	}
	if ingress.Hostname != "" {
		return ingress.Hostname, nil
	}
	return "", nil
}

func GetServiceResourceObjectKey(app *workload.KubernetesApplication) (client.ObjectKey, error) {
	for _, appResource := range app.Spec.ResourceTemplates {
		if appResource.Spec.Template.GetKind() == "Service" {
			return client.ObjectKey{Name: appResource.GetName(), Namespace: appResource.GetNamespace()}, nil
		}
	}
	return client.ObjectKey{}, fmt.Errorf("given KubernetesApplication %s does not have service type KubernetesApplicationResource", app.GetName())
}

func FillWithDefaults(wp *wordpressv1alpha1.WordpressInstance) {
	if wp.Spec.Image == "" {
		wp.Spec.Image = defaultWordpressImage
	}
}
