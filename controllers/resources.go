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
	"fmt"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	crossplanecore "github.com/crossplaneio/crossplane-runtime/apis/core/v1alpha1"
	compute "github.com/crossplaneio/crossplane/apis/compute/v1alpha1"
	database "github.com/crossplaneio/crossplane/apis/database/v1alpha1"
	workload "github.com/crossplaneio/crossplane/apis/workload/v1alpha1"

	"github.com/crossplaneio/sample-stack-wordpress/api/v1alpha1"
)

const (
	resourcePrefix               = "wordpress"
	localResourceSelectorKeyName = "wordpress-instance"
)

var (
	// Used by the resources in managed Kubernetes cluster where Wordpress
	// actually runs.
	labelsInRemote = map[string]string{
		"app": "wordpress",
	}
)

func ProduceKubernetesCluster(wp v1alpha1.WordpressInstance) *compute.KubernetesCluster {
	claimName := wp.GetName()
	return &compute.KubernetesCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:            claimName,
			Namespace:       wp.GetNamespace(),
			Labels:          GetLocalResourceSelector(wp),
			OwnerReferences: []metav1.OwnerReference{GetOwnerReference(wp)},
		},
		Spec: compute.KubernetesClusterSpec{
			ResourceClaimSpec: crossplanecore.ResourceClaimSpec{
				WriteConnectionSecretToReference: &crossplanecore.LocalSecretReference{
					Name: fmt.Sprintf("kubernetes-%s", claimName),
				},
			},
		},
	}
}

func ProduceMySQLInstance(wp v1alpha1.WordpressInstance) *database.MySQLInstance {
	claimName := wp.GetName()
	return &database.MySQLInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:            claimName,
			Namespace:       wp.GetNamespace(),
			Labels:          GetLocalResourceSelector(wp),
			OwnerReferences: []metav1.OwnerReference{GetOwnerReference(wp)},
		},
		Spec: database.MySQLInstanceSpec{
			ResourceClaimSpec: crossplanecore.ResourceClaimSpec{
				WriteConnectionSecretToReference: &crossplanecore.LocalSecretReference{
					Name: fmt.Sprintf("sql-%s", claimName),
				},
			},
			EngineVersion: "5.7",
		},
	}
}

func ProduceKubernetesApplication(scheme *runtime.Scheme, wp v1alpha1.WordpressInstance, dbConnectionSecretRef *crossplanecore.LocalSecretReference) (*workload.KubernetesApplication, error) {
	if dbConnectionSecretRef == nil {
		return nil, fmt.Errorf("cannot produce KubernetesApplication without local database connection secret")
	}
	namespaceInRemote, err := ConvertToUnstructured(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   wp.GetName(),
			Labels: labelsInRemote,
		},
	}, scheme)
	if err != nil {
		return nil, err
	}
	namespaceAppResourceInLocal := workload.KubernetesApplicationResourceTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-namespace-%s-%s", resourcePrefix, wp.GetNamespace(), wp.GetName()),
			Namespace: wp.GetNamespace(),
			Labels:    GetLocalResourceSelector(wp),
		},
		Spec: workload.KubernetesApplicationResourceSpec{
			Template: namespaceInRemote,
		},
	}

	// localDeploymentResourceName is needed in deploymentInRemote to generate
	// the valid secret name for database.
	localDeploymentResourceName := fmt.Sprintf("%s-deployment-%s-%s", resourcePrefix, wp.GetNamespace(), wp.GetName())
	deploymentInRemote, err := ConvertToUnstructured(&v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      wp.GetName(),
			Namespace: namespaceInRemote.GetName(),
			Labels:    labelsInRemote,
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labelsInRemote,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labelsInRemote,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "wordpress",
							Image: wp.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "wordpress",
									ContainerPort: int32(80),
								},
							},
							// The secret is propagated to the remote cluster with `localName-` prefix.
							Env: []corev1.EnvVar{
								{
									Name: "WORDPRESS_DB_HOST",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key: "endpoint",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: fmt.Sprintf("%s-%s", localDeploymentResourceName, dbConnectionSecretRef.Name),
											},
										},
									},
								},
								{
									Name: "WORDPRESS_DB_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key: "username",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: fmt.Sprintf("%s-%s", localDeploymentResourceName, dbConnectionSecretRef.Name),
											},
										},
									},
								},
								{
									Name: "WORDPRESS_DB_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key: "password",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: fmt.Sprintf("%s-%s", localDeploymentResourceName, dbConnectionSecretRef.Name),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, scheme)
	if err != nil {
		return nil, err
	}
	deploymentAppResourceInLocal := workload.KubernetesApplicationResourceTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      localDeploymentResourceName,
			Namespace: wp.GetNamespace(),
			Labels:    GetLocalResourceSelector(wp),
		},
		Spec: workload.KubernetesApplicationResourceSpec{
			Template: deploymentInRemote,
			Secrets: []corev1.LocalObjectReference{
				{Name: dbConnectionSecretRef.Name},
			},
		},
	}

	serviceInRemote, err := ConvertToUnstructured(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      wp.GetName(),
			Namespace: namespaceInRemote.GetName(),
			Labels:    labelsInRemote,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: int32(80),
				},
			},
			Selector: labelsInRemote,
			Type:     corev1.ServiceTypeLoadBalancer,
		},
	}, scheme)
	if err != nil {
		return nil, err
	}
	serviceAppResourceInLocal := workload.KubernetesApplicationResourceTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-service-%s-%s", resourcePrefix, wp.GetNamespace(), wp.GetName()),
			Namespace: wp.GetNamespace(),
			Labels:    GetLocalResourceSelector(wp),
		},
		Spec: workload.KubernetesApplicationResourceSpec{
			Template: serviceInRemote,
		},
	}
	return &workload.KubernetesApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:            wp.GetName(),
			Namespace:       wp.GetNamespace(),
			Labels:          GetLocalResourceSelector(wp),
			OwnerReferences: []metav1.OwnerReference{GetOwnerReference(wp)},
		},
		Spec: workload.KubernetesApplicationSpec{
			ResourceSelector: &metav1.LabelSelector{
				MatchLabels: GetLocalResourceSelector(wp),
			},
			TargetSelector: &metav1.LabelSelector{
				MatchLabels: GetLocalResourceSelector(wp),
			},
			ResourceTemplates: []workload.KubernetesApplicationResourceTemplate{
				namespaceAppResourceInLocal,
				deploymentAppResourceInLocal,
				serviceAppResourceInLocal,
			},
		},
	}, nil
}

func GetLocalResourceSelector(wp v1alpha1.WordpressInstance) map[string]string {
	return map[string]string{
		localResourceSelectorKeyName: wp.GetName(),
	}
}

func GetOwnerReference(wp v1alpha1.WordpressInstance) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: wp.APIVersion,
		Kind:       wp.Kind,
		Name:       wp.GetName(),
		UID:        wp.GetUID(),
	}
}

// ConvertToUnstructured takes a Kubernetes object and converts it into
// *unstructured.Unstructured that can be used as KubernetesApplication template.
// The reason metav1.Object is used instead of runtime.Object is that
// *unstructured.Unstructured requires the object to have metadata accessors.
func ConvertToUnstructured(o metav1.Object, scheme *runtime.Scheme) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	if err := scheme.Convert(o, u, nil); err != nil {
		return &unstructured.Unstructured{}, err
	}
	return u, nil
}
