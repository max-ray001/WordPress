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

	crossplanecore "github.com/crossplaneio/crossplane-runtime/apis/core/v1alpha1"
	compute "github.com/crossplaneio/crossplane/apis/compute/v1alpha1"
	database "github.com/crossplaneio/crossplane/apis/database/v1alpha1"
	workload "github.com/crossplaneio/crossplane/apis/workload/v1alpha1"

	"github.com/crossplaneio/sample-stack-wordpress/api/v1alpha1"
)

var (
	// Used by the resources in managed Kubernetes cluster where Wordpress
	// actually runs.
	labelsInRemote = map[string]string{
		"app": "wordpress",
	}
	// Used by the resources in the cluster where WordpressInstance custom
	// resource resides.
	labelsInLocal = map[string]string{
		"stack": "sample-stack-wordpress",
	}
)

func ProduceKubernetesCluster(wp v1alpha1.WordpressInstance) compute.KubernetesCluster {
	claimName := fmt.Sprintf("wordpress-cluster-%s-%s", wp.GetNamespace(), wp.GetName())
	return compute.KubernetesCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:            claimName,
			Namespace:       wp.GetNamespace(),
			Labels:          labelsInLocal,
			OwnerReferences: []metav1.OwnerReference{GetOwnerReference(wp)},
		},
		Spec: compute.KubernetesClusterSpec{
			ResourceClaimSpec: crossplanecore.ResourceClaimSpec{
				WriteConnectionSecretToReference: &crossplanecore.LocalSecretReference{
					Name: claimName,
				},
			},
		},
	}
}

func ProduceMySQLInstance(wp v1alpha1.WordpressInstance) database.MySQLInstance {
	claimName := fmt.Sprintf("wordpress-sql-%s-%s", wp.GetNamespace(), wp.GetName())
	return database.MySQLInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:            claimName,
			Namespace:       wp.GetNamespace(),
			Labels:          labelsInLocal,
			OwnerReferences: []metav1.OwnerReference{GetOwnerReference(wp)},
		},
		Spec: database.MySQLInstanceSpec{
			ResourceClaimSpec: crossplanecore.ResourceClaimSpec{
				WriteConnectionSecretToReference: &crossplanecore.LocalSecretReference{
					Name: claimName,
				},
			},
			EngineVersion: "5.7",
		},
	}
}

func ProduceKubernetesApplication(wp v1alpha1.WordpressInstance, dbClaimName string) workload.KubernetesApplication {
	namespaceResource := ConvertToUnstructured(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "wordpress",
			Labels: labelsInRemote,
		},
	})
	localDeploymentResourceName := fmt.Sprintf("wordpress-demo-deployment-%s-%s", wp.GetNamespace(), wp.GetName())
	return workload.KubernetesApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("wordpress-app-%s-%s", wp.GetNamespace(), wp.GetName()),
			Namespace:       wp.GetNamespace(),
			Labels:          labelsInLocal,
			OwnerReferences: []metav1.OwnerReference{GetOwnerReference(wp)},
		},
		Spec: workload.KubernetesApplicationSpec{
			ResourceSelector: &metav1.LabelSelector{
				MatchLabels: labelsInLocal,
			},
			ClusterSelector: &metav1.LabelSelector{
				MatchLabels: labelsInLocal,
			},
			ResourceTemplates: []workload.KubernetesApplicationResourceTemplate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   fmt.Sprintf("wordpress-demo-namespace-%s-%s", wp.GetNamespace(), wp.GetName()),
						Labels: labelsInLocal,
					},
					Spec: workload.KubernetesApplicationResourceSpec{
						Template: namespaceResource,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   localDeploymentResourceName,
						Labels: labelsInLocal,
					},
					Spec: workload.KubernetesApplicationResourceSpec{
						Template: ConvertToUnstructured(&v1.Deployment{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "wordpress",
								Namespace: namespaceResource.GetName(),
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
												Env: []corev1.EnvVar{
													{
														Name: "WORDPRESS_DB_HOST",
														ValueFrom: &corev1.EnvVarSource{
															SecretKeyRef: &corev1.SecretKeySelector{
																Key: "endpoint",
																LocalObjectReference: corev1.LocalObjectReference{
																	Name: fmt.Sprintf("%s-%s", localDeploymentResourceName, dbClaimName),
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
																	Name: fmt.Sprintf("%s-%s", localDeploymentResourceName, dbClaimName),
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
																	Name: fmt.Sprintf("%s-%s", localDeploymentResourceName, dbClaimName),
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
						}),
						Secrets: []corev1.LocalObjectReference{
							{Name: dbClaimName},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   fmt.Sprintf("wordpress-demo-service-%s-%s", wp.GetNamespace(), wp.GetName()),
						Labels: labelsInLocal,
					},
					Spec: workload.KubernetesApplicationResourceSpec{
						Template: ConvertToUnstructured(&corev1.Service{
							ObjectMeta: metav1.ObjectMeta{
								Name:      fmt.Sprintf("wordpress-demo-service-%s-%s", wp.GetNamespace(), wp.GetName()),
								Namespace: namespaceResource.GetName(),
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
						}),
					},
				},
			},
		},
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
