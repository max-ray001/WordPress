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
	"io"

	"strings"
	"text/template"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	wordpressv1alpha1 "github.com/crossplaneio/sample-stack-wordpress/api/v1alpha1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WordpressInstanceReconciler reconciles a WordpressInstance object
type WordpressInstanceReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=wordpress.samples.stacks.crossplane.io,resources=wordpressinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wordpress.samples.stacks.crossplane.io,resources=wordpressinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=compute.crossplane.io,resources=kubernetesclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.crossplane.io,resources=mysqlinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workload.crossplane.io,resources=kubernetesapplication,verbs=get;list;watch;create;update;patch;delete

func (r *WordpressInstanceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("wordpressinstance", req.NamespacedName)

	i := &wordpressv1alpha1.WordpressInstance{}
	if err := r.Client.Get(ctx, req.NamespacedName, i); err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	instanceUID := i.ObjectMeta.GetUID()

	rawTemplate := `---
apiVersion: v1
kind: Namespace
metadata:
  name: wordpresses
  labels:
    stack: sample-stack-wordpress
---
apiVersion: compute.crossplane.io/v1alpha1
kind: KubernetesCluster
metadata:
  name: wordpress-cluster-{{ .UID }}
  namespace: wordpresses
  labels:
    stack: sample-stack-wordpress
spec:
  writeConnectionSecretToRef:
    name: wordpress-demo-cluster-{{ .UID }}
---
apiVersion: database.crossplane.io/v1alpha1
kind: MySQLInstance
metadata:
  name: wordpress-mysql-{{ .UID }}
  namespace: wordpresses
  labels:
    stack: sample-stack-wordpress
spec:
  engineVersion: "5.7"
  # A secret is exported by providing the secret name
  # to export it under. This is the name of the secret
  # in the crossplane cluster, and it's scoped to this claim's namespace.
  writeConnectionSecretToRef:
    name: sql-{{ .UID }}
---
apiVersion: workload.crossplane.io/v1alpha1
kind: KubernetesApplication
metadata:
  name: wordpress-app-{{ .UID }}
  namespace: wordpresses
  labels:
    stack: sample-stack-wordpress
spec:
  resourceSelector:
    matchLabels:
      stack: sample-stack-wordpress
  clusterSelector:
    matchLabels:
      stack: sample-stack-wordpress
  resourceTemplates:
  - metadata:
      name: wordpress-demo-namespace-{{ .UID }}
      labels:
        stack: sample-stack-wordpress
    spec:
      template:
        apiVersion: v1
        kind: Namespace
        metadata:
          name: wordpress
          labels:
            app: wordpress
  - metadata:
      name: wordpress-demo-deployment-{{ .UID }}
      labels:
        stack: sample-stack-wordpress
    spec:
      secrets:
        # This must match the writeConnectionSecretToRef field
        # on the database claim; it is the name of the secret to
        # pull from the crossplane cluster, from this Application's namespace.
        - name: sql-{{ .UID }}
      template:
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          namespace: wordpress
          name: wordpress
          labels:
            app: wordpress
        spec:
          selector:
            matchLabels:
              app: wordpress
          template:
            metadata:
              labels:
                app: wordpress
            spec:
              containers:
                - name: wordpress
                  image: wordpress:4.6.1-apache
                  env:
                    - name: WORDPRESS_DB_HOST
                      valueFrom:
                        secretKeyRef:
                          # This is the name of the secret to use to consume the secret
                          # within the managed cluster. The reason it's different from the
                          # name of the secret above is because within the managed cluster,
                          # a crossplane-managed secret is written as '{metadata.name}-{secretname}'.
                          # The metadata name is specified above for this resource, and so is
                          # the secret name.
                          name: wordpress-demo-deployment-{{ .UID }}-sql-{{ .UID }}
                          key: endpoint
                    - name: WORDPRESS_DB_USER
                      valueFrom:
                        secretKeyRef:
                          name: wordpress-demo-deployment-{{ .UID }}-sql-{{ .UID }}
                          key: username
                    - name: WORDPRESS_DB_PASSWORD
                      valueFrom:
                        secretKeyRef:
                          name: wordpress-demo-deployment-{{ .UID }}-sql-{{ .UID }}
                          key: password
                  ports:
                    - containerPort: 80
                      name: wordpress
  - metadata:
      name: wordpress-demo-service-{{ .UID }}
      labels:
        stack: sample-stack-wordpress
    spec:
      template:
        apiVersion: v1
        kind: Service
        metadata:
          namespace: wordpress
          name: wordpress
          labels:
            app: wordpress
        spec:
          ports:
            - port: 80
          selector:
            app: wordpress
          type: LoadBalancer
`
	r.Log.V(2).Info("Using template", "template", rawTemplate)

	tmpl, err := template.New("wordpress").Parse(rawTemplate)
	if err != nil {
		r.Log.V(0).Info("Error creating template!", "err", err)
	}

	var sb strings.Builder

	data := map[string]interface{}{
		"UID": instanceUID,
	}

	tmpl.Execute(&sb, data)

	tmplOutput := sb.String()

	r.Log.V(2).Info("Using yaml", "yaml", tmplOutput)

	// TODO
	// Return a reconcile error when there's an issue
	// Set instance owner as the Stack?
	// Add labels to resources created by individual wordpress instance, and put that
	//     same label on the instance. For later querying
	// Maybe put the IP of the load balancer in the wordpress instance CRD
	// Represent the status of the instance in the CRD?
	// Use defaults
	r.Log.V(2).Info("BEFORE extract objects")
	objects, err := r.ExtractObjects(ctx, &tmplOutput)
	if err != nil {
		r.Log.V(0).Info("Error extracting objects!", "err", err)
	}

	err = r.createObjects(ctx, objects, i)

	if err != nil {
		r.Log.V(0).Info("Error creating objects!", "err", err)
	}

	return ctrl.Result{}, nil
}

func (r *WordpressInstanceReconciler) ExtractObjects(ctx context.Context, s *string) ([]*unstructured.Unstructured, error) {
	// read full output from job by retrieving the logs for the job's pod
	r.Log.V(2).Info("ENTER extract objects", "ctx", ctx, "string", s)
	reader := strings.NewReader(*s)

	// decode and process all resources from job output
	d := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	var objects []*unstructured.Unstructured
	for {
		obj := &unstructured.Unstructured{}
		if err := d.Decode(&obj); err != nil {
			if err == io.EOF {
				// we reached the end of the job output
				r.Log.V(2).Info("EXIT extract objects because EOF", "objects", objects, "err", err)
				break
			}
			r.Log.V(2).Info("EXIT extract objects because ERROR", "objects", objects, "err", err)
			return nil, errors.Wrapf(err, "failed to parse output")
		}

		objects = append(objects, obj)
	}

	r.Log.V(2).Info("EXIT extract objects", "objects", objects)
	return objects, nil
}

func (r *WordpressInstanceReconciler) createObjects(ctx context.Context, objects []*unstructured.Unstructured, i *wordpressv1alpha1.WordpressInstance) error {
	for _, obj := range objects {
		// process and create the object that we just decoded
		if err := r.createOutputObject(ctx, obj, i); err != nil {
			return err
		}
	}

	return nil
}

func (r *WordpressInstanceReconciler) createOutputObject(ctx context.Context, obj *unstructured.Unstructured, i *wordpressv1alpha1.WordpressInstance) error {
	// if we decoded a non-nil unstructured object, try to create it now
	if obj == nil {
		return nil
	}

	// set an owner reference on the object
	obj.SetOwnerReferences([]metav1.OwnerReference{
		AsOwner(ReferenceTo(i, wordpressv1alpha1.WordpressInstanceGroupVersionKind)),
	})

	r.Log.V(1).Info(
		"creating object",
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
		"apiVersion", obj.GetAPIVersion(),
		"kind", obj.GetKind(),
		"ownerRefs", obj.GetOwnerReferences())

	if err := r.Client.Create(ctx, obj); err != nil && !kerrors.IsAlreadyExists(err) {
		return errors.Wrapf(err, "failed to create object %s: %s", obj.GetName(), err)
	}

	return nil
}

func (r *WordpressInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wordpressv1alpha1.WordpressInstance{}).
		Complete(r)
}

// HACK:
// The utility methods below have been copied from Crossplane (https://github.com/crossplaneio/crossplane)

// ReferenceTo returns an object reference to the supplied object, presumed to
// be of the supplied group, version, and kind.
func ReferenceTo(o metav1.Object, of schema.GroupVersionKind) *corev1.ObjectReference {
	v, k := of.ToAPIVersionAndKind()
	return &corev1.ObjectReference{
		APIVersion: v,
		Kind:       k,
		Namespace:  o.GetNamespace(),
		Name:       o.GetName(),
		UID:        o.GetUID(),
	}
}

// AsOwner converts the supplied object reference to an owner reference.
func AsOwner(r *corev1.ObjectReference) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: r.APIVersion,
		Kind:       r.Kind,
		Name:       r.Name,
		UID:        r.UID,
	}
}
