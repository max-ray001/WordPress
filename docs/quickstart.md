# Wordpress Quick Start
This guide walks through deploying the Wordpress application using Crossplane
using the [templating-controller].

## Overview
 - [Setup Environment]
 - [Install Provider and Stack]
 - [Install Application]
 - [Cleanup]
 - [Debugging]

## Setup Environment
### Create a Crossplane Environment
* Create a Crossplane environment as your control plane for your apps and
infrastructure. 
  * To setup an environment by hand, [install crossplane] from the
alpha channel.

* Create an application namespace `workspace1` in your Crossplane environment.
  * `kubectl create namespace workspace1`

### Install the Crossplane CLI
```
curl -sL https://raw.githubusercontent.com/crossplane/crossplane-cli/master/bootstrap.sh | bash
```
See [Crossplane CLI] for details.

## Install Provider and Stack
Wordpress can be deployed on top of any stack that provides the default
resource classes it needs as part of its self-service catalog.

This guide walks through 3 cloud provider options:
 - [GCP Provider and Stack]
 - [AWS Provider and Stack]
 - [Azure Provider and Stack]

 Select one of the above and then continue to [Install Application].

### GCP Provider and Stack


#### Install GCP Provider
```
# optional alternate registry
# REGISTRY=registry.upbound.io

PACKAGE=crossplane/provider-gcp:v0.10.0
NAME=provider-gcp
kubectl crossplane package install --cluster --namespace crossplane-system ${PACKAGE} ${NAME} ${REGISTRY}
```

#### Install GCP Sample Stack
```
# optional alternate registry
# REGISTRY=registry.upbound.io

PACKAGE=crossplane/stack-gcp-sample:v0.6.0
NAME=stack-gcp-sample
kubectl crossplane package install --cluster --namespace crossplane-system ${PACKAGE} ${NAME} ${REGISTRY}
```

See [GCP Sample Stack] for details.

#### Verify packages installed and ready
```
kubectl get clusterpackageinstall -A
```

and wait for them to be `Ready:True`
```
NAMESPACE           NAME               READY   SOURCE                PACKAGE
crossplane-system   provider-gcp       True    registry.upbound.io   crossplane/provider-gcp:v0.10.0
crossplane-system   stack-gcp-sample   True    registry.upbound.io   crossplane/stack-gcp-sample:v0.6.0
```

#### Get GCP Account Keyfile
```
# replace this with your own gcp project id and service account name
PROJECT_ID=my-project
SA_NAME=my-service-account-name

# create service account
SA="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" 
gcloud iam service-accounts create $SA_NAME --project $PROJECT_ID

# enable cloud APIs
gcloud services enable container.googleapis.com --project $PROJECT_ID
gcloud services enable sqladmin.googleapis.com --project $PROJECT_ID
gcloud services enable compute.googleapis.com --project $PROJECT_ID
gcloud services enable servicenetworking.googleapis.com --project $PROJECT_ID

# grant access
gcloud projects add-iam-policy-binding --role="roles/iam.serviceAccountUser" $PROJECT_ID --member "serviceAccount:$SA"
gcloud projects add-iam-policy-binding --role="roles/cloudsql.admin" $PROJECT_ID --member "serviceAccount:$SA"
gcloud projects add-iam-policy-binding --role="roles/container.admin" $PROJECT_ID --member "serviceAccount:$SA"
gcloud projects add-iam-policy-binding --role="roles/compute.networkAdmin" $PROJECT_ID --member "serviceAccount:$SA"

# create service account keyfile
gcloud iam service-accounts keys create creds.json --project $PROJECT_ID --iam-account $SA
```

#### Create a Provider Secret
```
kubectl create secret generic gcp-creds -n crossplane-system --from-file=key=./creds.json
```

#### Create and apply stack.yaml

Create `stack.yaml` replacing the GCP `PROJECT_ID` with your own:
```
cat > stack.yaml <<EOF
apiVersion: gcp.stacks.crossplane.io/v1alpha1
kind: GCPSample
metadata:
  name: my-cool-stack
spec:
  # replace this with your own gcp project id
  projectID: ${PROJECT_ID}
  region: us-west1
  credentialsSecretRef:
    namespace: crossplane-system
    name: gcp-creds
    key: key
EOF
```

Apply `stack.yaml`
```
kubectl apply -f stack.yaml
```

Verify the `GCPSample` resource was created and has a `status.conditions` of `Synced` and `status:True`.
```
kubectl get gcpsample -A -o yaml
```

Which should show:
```
apiVersion: v1
items:
- apiVersion: gcp.stacks.crossplane.io/v1alpha1
  kind: GCPSample
  metadata:
    annotations:
      kubectl.kubernetes.io/last-applied-configuration: |
        {"apiVersion":"gcp.stacks.crossplane.io/v1alpha1","kind":"GCPSample","metadata":{"annotations":{},"name":"my-cool-stack"},"spec":{"credentialsSecretRef":{"key":"key","name":"gcp-creds","namespace":"crossplane-system"},"projectID":"crossplane-playground","region":"us-west1"}}
    creationTimestamp: "2020-05-21T00:40:58Z"
    finalizers:
    - templating-controller.crossplane.io
    generation: 1
    name: my-cool-stack
    resourceVersion: "4519473"
    selfLink: /apis/gcp.stacks.crossplane.io/v1alpha1/gcpsamples/my-cool-stack
    uid: 614c8c29-c472-4829-bf0b-4f2e952ed9ef
  spec:
    credentialsSecretRef:
      key: key
      name: gcp-creds
      namespace: crossplane-system
    projectID: crossplane-playground
    region: us-west1
  status:
    conditions:
    - lastTransitionTime: "2020-05-21T00:41:03Z"
      reason: Successfully reconciled resource
      status: "True"
      type: Synced
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

#### Next Steps: Deploy an Application
Once the GCP `Stack` is installed and configured skip to [Install Application].

### AWS Provider and Stack
#### Install AWS Provider
```
# optional alternate registry
# REGISTRY=registry.upbound.io

PACKAGE=crossplane/provider-aws:v0.10.0
NAME=provider-aws
kubectl crossplane package install --cluster --namespace crossplane-system ${PACKAGE} ${NAME} ${REGISTRY}
```

#### Install AWS Sample Stack
```
# optional alternate registry
# REGISTRY=registry.upbound.io

PACKAGE=crossplane/stack-aws-sample:v0.6.0
NAME=stack-aws-sample
kubectl crossplane package install --cluster --namespace crossplane-system ${PACKAGE} ${NAME} ${REGISTRY}
```

See [AWS Sample Stack] for details.

#### Verify packages installed and ready
```
kubectl get clusterpackageinstall -A
```

and wait for them to be `Ready:True`
```
NAMESPACE           NAME               READY   SOURCE                PACKAGE
crossplane-system   provider-aws       False   registry.upbound.io   crossplane/provider-aws:v0.10.0
crossplane-system   stack-aws-sample   False   registry.upbound.io   crossplane/stack-aws-sample:v0.6.0
```

#### Get AWS IAM Keyfile
Using an AWS account with permissions to manage AKS, RDS, networking, and identity resources in [stack-aws-sample](https://github.com/crossplane/stack-aws-sample/tree/master/kustomize/aws):
```
AWS_PROFILE=default && echo -e "[default]\naws_access_key_id = $(aws configure get aws_access_key_id --profile $AWS_PROFILE)\naws_secret_access_key = $(aws configure get aws_secret_access_key --profile $AWS_PROFILE)" > creds.conf
```

#### Create a Provider Secret
```
kubectl create secret generic aws-creds -n crossplane-system --from-file=key=./creds.conf
```

#### Create and apply stack.yaml

Create `stack.yaml`: 
```
apiVersion: aws.stacks.crossplane.io/v1alpha1
kind: AWSSample
metadata:
  name: my-cool-stack
spec:
  region: us-west-2
  credentialsSecretRef:
    key: key
    name: aws-creds
    namespace: crossplane-system
```

Apply `stack.yaml`:
```
kubectl apply -f stack.yaml
```

Verify the `AWSSample` resource was created and has a `status.conditions` of `Synced` and `status:True`.
```
kubectl get awssample -A -o yaml
```

Which should show:
```
apiVersion: v1
items:
- apiVersion: aws.stacks.crossplane.io/v1alpha1
  kind: AWSSample
  metadata:
    annotations:
      kubectl.kubernetes.io/last-applied-configuration: |
        {"apiVersion":"aws.stacks.crossplane.io/v1alpha1","kind":"AWSSample","metadata":{"annotations":{},"name":"my-cool-stack"},"spec":{"credentialsSecretRef":{"key":"key","name":"aws-creds","namespace":"crossplane-system"},"region":"us-west-2"}}
    creationTimestamp: "2020-05-21T00:55:25Z"
    finalizers:
    - templating-controller.crossplane.io
    generation: 1
    name: my-cool-stack
    resourceVersion: "4527961"
    selfLink: /apis/aws.stacks.crossplane.io/v1alpha1/awssamples/my-cool-stack
    uid: e6e2b315-a763-4179-8304-b2f2a42d50d8
  spec:
    credentialsSecretRef:
      key: key
      name: aws-creds
      namespace: crossplane-system
    region: us-west-2
  status:
    conditions:
    - lastTransitionTime: "2020-05-21T00:55:30Z"
      reason: Successfully reconciled resource
      status: "True"
      type: Synced
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

#### Next Steps: Deploy an Application
Once the AWS `Stack` is installed and configured skip to [Install Application].

### Azure Provider and Stack
#### Install Azure Provider
```
# optional alternate registry
# REGISTRY=registry.upbound.io

PACKAGE=crossplane/provider-azure:v0.10.0
NAME=provider-azure
kubectl crossplane package install --cluster --namespace crossplane-system ${PACKAGE} ${NAME} ${REGISTRY}
```

#### Install Azure Sample Stack
```
# optional alternate registry
# REGISTRY=registry.upbound.io

PACKAGE=crossplane/stack-azure-sample:v0.6.0
NAME=stack-azure-sample
kubectl crossplane package install --cluster --namespace crossplane-system ${PACKAGE} ${NAME} ${REGISTRY}
```

See [Azure Sample Stack] for details.

#### Verify packages installed and ready
```
kubectl get clusterpackageinstall -A
```

and wait for them to be `Ready:True`
```
NAMESPACE           NAME                 READY   SOURCE                PACKAGE
crossplane-system   provider-azure       True    registry.upbound.io   crossplane/provider-azure:v0.10.0
crossplane-system   stack-azure-sample   True    registry.upbound.io   crossplane/stack-azure-sample:v0.6.0
```

#### Get Azure Principal Keyfile
```
# create service principal with Owner role
az ad sp create-for-rbac --sdk-auth --role Owner > "creds.json"

# add Azure Active Directory permissions
AZURE_CLIENT_ID=$(jq -r ".clientId" < "./creds.json")

RW_ALL_APPS=1cda74f2-2616-4834-b122-5cb1b07f8a59
RW_DIR_DATA=78c8a3c8-a07e-4b9e-af1b-b5ccab50a175
AAD_GRAPH_API=00000002-0000-0000-c000-000000000000

az ad app permission add --id "${AZURE_CLIENT_ID}" --api ${AAD_GRAPH_API} --api-permissions ${RW_ALL_APPS}=Role ${RW_DIR_DATA}=Role
az ad app permission grant --id "${AZURE_CLIENT_ID}" --api ${AAD_GRAPH_API} --expires never > /dev/null
az ad app permission admin-consent --id "${AZURE_CLIENT_ID}"
```

#### Create a Provider Secret
```
kubectl create secret generic azure-creds -n crossplane-system --from-file=key=./creds.json
```

#### Create and apply stack.yaml
Create `stack.yaml`:
```
apiVersion: azure.stacks.crossplane.io/v1alpha1
kind: AzureSample
metadata:
  name: my-cool-stack
spec:
  region: us-west-2
  credentialsSecretRef:
    key: key
    name: azure-creds
    namespace: crossplane-system
```

Apply `stack.yaml`:
```
kubectl apply -f stack.yaml
```

Verify the `AzureSample` resource was created and has a `status.conditions` of `Synced` and `status:True`.
```
kubectl get azuresample -A -o yaml
```

Which should show:
```
apiVersion: v1
items:
- apiVersion: azure.stacks.crossplane.io/v1alpha1
  kind: AzureSample
  metadata:
    annotations:
      kubectl.kubernetes.io/last-applied-configuration: |
        {"apiVersion":"azure.stacks.crossplane.io/v1alpha1","kind":"AzureSample","metadata":{"annotations":{},"name":"my-cool-stack"},"spec":{"credentialsSecretRef":{"key":"key","name":"azure-creds","namespace":"crossplane-system"},"region":"us-west-2"}}
    creationTimestamp: "2020-05-21T17:50:30Z"
    finalizers:
    - templating-controller.crossplane.io
    generation: 1
    name: my-cool-stack
    resourceVersion: "5366350"
    selfLink: /apis/azure.stacks.crossplane.io/v1alpha1/azuresamples/my-cool-stack
    uid: 627d3d19-777d-4690-9094-92a1483eb55c
  spec:
    credentialsSecretRef:
      key: key
      name: azure-creds
      namespace: crossplane-system
    region: us-west-2
  status:
    conditions:
    - lastTransitionTime: "2020-05-21T17:50:34Z"
      reason: Successfully reconciled resource
      status: "True"
      type: Synced
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

Note: Azure requires an [Azure MySQLServerVirtualNetworkRule] to be created *after* the `MySqlInstance` claim is bound as part of deploying the `Application` instance. A new [Composition feature](https://github.com/crossplane/crossplane/issues/1343) will remove this extra step. See [Azure MySQLServerVirtualNetworkRule] for details.

#### Next Steps: Deploy an Application
Once the Azure `Stack` is installed and configured skip to [Install Application].

## Install Application
### Install the Wordpress Application
```
# optional alternate registry
# REGISTRY=registry.upbound.io

PACKAGE=crossplane/app-wordpress:v0.5.0
NAMESPACE=workspace1
NAME=app-wordpress
kubectl crossplane package install --namespace ${NAMESPACE} ${PACKAGE} ${NAME} ${REGISTRY}
```

### Verify packages installed and ready
```
kubectl get packageinstall -A
```

and wait for it to be `Ready:True`
```
NAMESPACE    NAME            READY   SOURCE                PACKAGE
workspace1   app-wordpress   True    registry.upbound.io   crossplane/app-wordpress:v0.5.0
```

### Create and apply app.yaml
Create `app.yaml`:
```
apiVersion: wordpress.apps.crossplane.io/v1alpha1
kind: WordpressInstance
metadata:
  namespace: workspace1
  name: my-cool-app
spec:
  provisionPolicy: ProvisionNewCluster
```

Then apply it:
```
kubectl apply -f app.yaml
```

Verify the `WordpressInstance` was created:
```
kubectl get -A wordpressinstance
```

which should show:
```
NAME          AGE
my-cool-app   78m
```

Then watch for the `KubernetesCluster` and `MySQLInstance` to become bound:
```
kubectl get -A kubernetescluster
kubectl get -A mysqlinstance
```

These results show `Wordpress` deployed on top of the `GCP Sample Stack` and being bound to the `GKEClusterClass` and `CloudSQLInstanceClass`:
```
NAME                  STATUS   CLASS-KIND              CLASS-NAME                                  RESOURCE-KIND      RESOURCE-NAME                          AGE
my-cool-app-cluster            GKEClusterClass         my-cool-stack-gkeclusterclass               GKECluster         workspace1-my-cool-app-cluster-swh68   8m
my-cool-app-sql       Bound    CloudSQLInstanceClass   my-cool-stack-cloudsqlinstanceclass-mysql   CloudSQLInstance   workspace1-my-cool-app-sql-cfqhz       8m
```

The `KubernetesCluster` above is not `Bound` yet and the provisioning status is found on the underlying managed resource:
```
GROUP=$(kubectl get kubernetescluster -n workspace1 -o=jsonpath='{.items[0].spec.resourceRef.apiVersion}' | awk -F'/' '{print $1}')

KIND=$(kubectl get kubernetescluster -n workspace1 -o=jsonpath='{.items[0].spec.resourceRef.kind}')

NAME=$(kubectl get kubernetescluster -n workspace1 -o=jsonpath='{.items[0].spec.resourceRef.name}')

kubectl get ${KIND}.${GROUP} ${NAME} -o yaml
```

During provisioning there will be a `status.condition` of `Ready: False`.
```
status:
  conditions:
  - lastTransitionTime: "2020-04-30T19:07:55Z"
    reason: Resource is being created
    status: "False"
    type: Ready
```

The `MySQLInstance` provisioning status is found on the underlying managed resource:
```
GROUP=$(kubectl get mysqlinstance -n workspace1 -o=jsonpath='{.items[0].spec.resourceRef.apiVersion}' | awk -F'/' '{print $1}')

KIND=$(kubectl get mysqlinstance -n workspace1 -o=jsonpath='{.items[0].spec.resourceRef.kind}')

NAME=$(kubectl get mysqlinstance -n workspace1 -o=jsonpath='{.items[0].spec.resourceRef.name}')

kubectl get ${KIND}.${GROUP} ${NAME} -o yaml
```

Successful provisoining results in `bindingPhase: Bound` and a `status.condition` of `Ready: True`
```
status:
  bindingPhase: Bound
  conditions:
  - lastTransitionTime: "2020-04-30T19:12:11Z"
    reason: Resource is available for use
    status: "True"
    type: Ready
```

The `KubernetesApplication` contains the `Deployment`, `Namespace`, and `Service` to be deployed to the remote `KubernetesTarget` cluster:
```
kubectl get kubernetesapplication -A -o yaml
```

In summary:
```
spec:
    resourceTemplates:
      spec:
        template:
          apiVersion: v1
          kind: Namespace
          metadata:
            labels:
              app: wordpress
            name: my-cool-app
      spec:
        secrets:
        - name: sql
        template:
          apiVersion: apps/v1
          kind: Deployment
      spec:
        template:
          apiVersion: v1
          kind: Service
          metadata:
            labels:
              app: wordpress
            name: wordpress
            namespace: my-cool-app
          spec:
            ports:
            - port: 80
            selector:
              app: wordpress
            type: LoadBalancer
    targetRef:
      name: my-cool-app-cluster
    targetSelector:
      matchLabels:
        app: my-cool-app
```

Once the `KubernetesCluster` `status` becomes `Bound`, the `KubernetesApplication` will be scheduled to the `KubernetesTarget` and create `KubernetesApplicationResources` (KAR) resources that are control plane representations of the actual `Namespace`, `Deployment`, and `Service` resources that will be created on the remote `KubernetesCluster` when it becomes available.

During provisioning these KAR resources will have the following error `status`
waiting for the `KubernetesTarget` to become available. 
```
kubectl get kubernetesapplicationresources -A -o yaml
```

```
  status:
    conditionedStatus:
      conditions:
      - lastTransitionTime: "2020-04-29T23:14:08Z"
        message: failed to find a usable KubernetesTarget for scheduling
        reason: Encountered an error during resource reconciliation
        status: "False"
        type: Synced
    state: Pending
```

This is normal and Crossplane's continous reconcilation will automatically
resolve this when the target `KubernetesCluster` becomes available.

When the `KubernetesApplication` has been deployed the KAR Service resource
will have the `loadBalancer` IP:
```
kubectl get kubernetesapplicationresources -A -o yaml
```
```
status:
   remote:
      loadBalancer:
        ingress:
        - ip: 34.83.28.96
```

Which shows that `Wordpress` has been deployed at the IP above onto a dynamically provisioned `KubernetesCluster` and `MySQLInstance`:
![installed](installed.png)

Note: Azure requires an [Azure MySQLServerVirtualNetworkRule] to be created *after* the `MySqlInstance` claim is bound as part of deploying the `Application` instance. A new [Composition feature](https://github.com/crossplane/crossplane/issues/1343) will remove this extra step. See [Azure MySQLServerVirtualNetworkRule] for details.

If you run into other issues with your Wordpress deployment see [Debugging].

## Cleanup
### Delete the application instance:
```
kubectl delete wordpressinstance -n workspace1 my-cool-app
```

### Delete the stack instance, using the appropriate command for your cloud provider:
```
kubectl delete gcpsample my-cool-stack

kubectl delete awssample my-cool-stack

kubectl delete azuresample my-cool-stack
```

#### Wait for the `Network` / `VCP` to be cleaned up.
```
kubectl get networks.compute.gcp.crossplane.io
kubectl get subnetworks.compute.gcp.crossplane.io

kubectl get vpcs.network.aws.crossplane.io
kubectl get subnets.network.aws.crossplane.io

kubectl get resourcegroup.azure.crossplane.io
kubectl get virtualnetworks.network.azure.crossplane.io
kubectl get subnets.network.azure.crossplane.io
```

### Uninstall Application
```
NAME=app-wordpress
kubectl crossplane package uninstall --namespace workspace1 ${NAME}
```

### Uninstall Stack
Uninstall the Stack with one of the following:
```
NAME=stack-gcp-sample
kubectl crossplane package uninstall --cluster --namespace crossplane-system ${NAME}
```
```
NAME=stack-aws-sample
kubectl crossplane package uninstall --cluster --namespace crossplane-system ${NAME}
```
```
NAME=stack-azure-sample
kubectl crossplane package uninstall --cluster --namespace crossplane-system ${NAME}
```

### Uninstall Provider
Uninstall the Provider with one of the following:
```
NAME=provider-gcp
kubectl crossplane package uninstall --cluster --namespace crossplane-system ${NAME}
```
```
NAME=provider-aws
kubectl crossplane package uninstall --cluster --namespace crossplane-system ${NAME}
```
```
NAME=provider-azure
kubectl crossplane package uninstall --cluster --namespace crossplane-system ${NAME}
```

### Delete the crossplane-system namespace
```
kubectl delete ns crossplane-system
```

# Done!

## Debugging
Wordpress doesn't deploy as expected:
 - [MySQLInstance is Ready]
 - [KubernetesCluster is Ready]
 - [Deployment and Service Debugging]
 - [Azure MySQLServerVirtualNetworkRule]

Networking resources don't delete successfully:
 - [AWS Networking DependencyViolation]

### MySQLInstance is Ready
Verify the MySQLInstance is bound with a status condition of Ready: true.

If the `MySQLInstance` is not `Bound` the underlying managed resource will have a `status` with more details.
```
kubectl get -A mysqlinstance -o yaml
```

In a GCP deployment the `MySQLInstance` `spec.resourceRef` will be set to a `CloudSQLInstance`:
```
spec:
    classRef:
      apiVersion: database.gcp.crossplane.io/v1beta1
      kind: CloudSQLInstanceClass
      name: my-cool-stack-cloudsqlinstanceclass-mysql
      uid: 82704895-9ac1-4706-9383-797e5f61006f
    engineVersion: "5.7"
    resourceRef:
      apiVersion: database.gcp.crossplane.io/v1beta1
      kind: CloudSQLInstance
      name: workspace1-my-cool-app-sql-cfqhz
      uid: 81c00371-08f8-4c2a-aa09-f2b1386a0e71
    writeConnectionSecretToRef:
      name: sql
```

The underlying managed resource will have additional `status`:
```
kubectl get -A cloudsqlinstance.database.gcp.crossplane.io -o yaml
```

A successful results has a `spec.status.condition` of `type: Ready` and `status: True`
```
status:
    bindingPhase: Bound
    conditions:
    - lastTransitionTime: "2020-04-29T17:57:32Z"
      reason: Resource is available for use
      status: "True"
      type: Ready
```

Any errors will surface in the above `status` and are typically related to errors in the underlying cloud provider APIs.

### KubernetesCluster is Ready
Verify KubernetesCluster is bound with a status condition of Ready: true.

If the `KubernetesCluster` is not `Bound` the underlying managed resource will have a `status` with more details.
```
kubectl get -A kubernetescluster -o yaml
```

in a GCP deployment the `KubernetesCluster` `spec.resourceRef` will be set to a `GCKCluster`
```
spec:
    classRef:
      apiVersion: compute.gcp.crossplane.io/v1alpha3
      kind: GKEClusterClass
      name: my-cool-stack-gkeclusterclass
      uid: cb4eda60-90c1-48bd-a0f1-b122c6224565
    resourceRef:
      apiVersion: compute.gcp.crossplane.io/v1alpha3
      kind: GKECluster
      name: workspace1-my-cool-app-cluster-swh68
      uid: e5d132d7-db70-4ab8-9892-77ea1e097d72
    writeConnectionSecretToRef:
      name: my-cool-app-cluster
```

The underlying managed resource will have additional `status`:
```
kubectl get -A gkecluster.compute.gcp.crossplane.io -o yaml
```

A successful results has a `spec.status.condition` of `type: Ready` and `status: True`
```
status:
    bindingPhase: Bound
    conditions:
    - lastTransitionTime: "2020-04-29T17:57:32Z"
      reason: Resource is available for use
      status: "True"
      type: Ready
```

Any errors will surface in the above `status` and are typically related to errors in the underlying cloud provider APIs.

### Deployment and Service Debugging
Verify Deployment and Service have been created on the remote KubernetesTarget.

If the 'MySQLInstance' and the `KubernetesCluster` have been created, the next
thing to check is that the `Deployment` and `Service` have been created on the
remote `KubernetesTarget` and the container logs look correct.

Create a `remote.kubeconfig` for the remote `KubernetesTarget`:
```
SECRET_NAME=$(kubectl get kubernetestarget -n workspace1 -o=jsonpath='{.items[0].spec.connectionSecretRef.name}')
kubectl get -n workspace1 secret ${SECRET_NAME} --template={{.data.kubeconfig}} | base64 --decode > remote.kubeconfig
```

Verify the following:

`Namespace` `my-cool-app` from the `KubernetesApplication` shows up on the `KubernetesTarget`:
```
kubectl --kubeconfig=remote.kubeconfig get namespaces
```

```
NAME              STATUS   AGE
default           Active   173m
kube-node-lease   Active   173m
kube-public       Active   173m
kube-system       Active   173m
my-cool-app       Active   172m
```

`Deployment` `wordpress` from the `KubernetesApplication` shows up on the `KubernetesTarget`:
```
kubectl --kubeconfig=remote.kubeconfig get -n my-cool-app deployments
```

```
NAME        READY   UP-TO-DATE   AVAILABLE   AGE
wordpress   1/1     1            1           3h2m
```

`Pod` `wordpress-<suffix>` shows up on the `KubernetesTarget`:
```
kubectl --kubeconfig=remote.kubeconfig get -n my-cool-app pods
```
```
NAME                         READY   STATUS    RESTARTS   AGE
wordpress-6d6fd567d4-4zn85   1/1     Running   0          3
```

`Pod` logs look correct and is serving traffic:
```
POD=$(kubectl --kubeconfig=remote.kubeconfig get pods -n my-cool-app -o name)
kubectl --kubeconfig=remote.kubeconfig logs -n my-cool-app ${POD}
```
```
WordPress not found in /var/www/html - copying now...
Complete! WordPress has been successfully copied to /var/www/html
10.128.0.1 - - [29/Apr/2020:19:34:43 +0000] "GET /favicon.ico HTTP/1.1" 200 228 "http://34.83.28.96/wp-admin/install.php" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.129 Safari/537.36"
10.128.0.1 - - [29/Apr/2020:20:52:12 +0000] "GET / HTTP/1.1" 302 323 "-" "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36"
10.128.0.1 - - [29/Apr/2020:20:57:26 +0000] "GET / HTTP/1.1" 302 320 "-" "-"
```

`Service` `wordpress` is available with an `EXTERNAL-IP`:
```
kubectl --kubeconfig=remote.kubeconfig get services -n my-cool-app
```
```
NAME        TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)        AGE
wordpress   LoadBalancer   172.16.158.197   34.83.28.96   80:32599/TCP   3h13m
```

### Azure MySQLServerVirtualNetworkRule
Ensure the Azure `MySQLServerVnetRule` is created to open ports to individual `MySQLInstances`.

Azure requires a `MySQLServerVnetRule` to be created *after* the
`MySqlInstance` claim is bound as part of deploying the `Application` instance.
A new [Composition feature](https://github.com/crossplane/crossplane/issues/1343) 
will remove this extra step.

Run the following to generate the `vnetrule.yaml`:
```
SERVER_NAME=$(kubectl get mysqlservers -o=jsonpath='{.items[0].metadata.name}')
RES_GROUP_NAME=$(kubectl get mysqlservers -o=jsonpath='{.items[0].spec.forProvider.resourceGroupName}')
SUBNET_NAME=$(kubectl get subnets -o=jsonpath='{.items[0].metadata.name}')
PROVIDER_NAME=$(kubectl get mysqlservers -o=jsonpath='{.items[0].spec.providerRef.name}')

cat > vnetrule.yaml <<EOF
apiVersion: database.azure.crossplane.io/v1alpha3
kind: MySQLServerVirtualNetworkRule
metadata:
  name: my-cool-app-vnetrule
spec:
  serverName: ${SERVER_NAME}
  resourceGroupNameRef:
    name: ${RES_GROUP_NAME}
  properties:
    virtualNetworkSubnetIdRef:
      name: ${SUBNET_NAME}
  reclaimPolicy: Delete
  providerRef:
    name: ${PROVIDER_NAME}
EOF
```

The `vnetrule.yaml` should look like this:
```
apiVersion: database.azure.crossplane.io/v1alpha3
kind: MySQLServerVirtualNetworkRule
metadata:
  name: my-cool-app-vnetrule
spec:
  serverName: workspace1-my-cool-app-sql-jmlnc
  resourceGroupNameRef:
    name: my-cool-stack-resourcegroup
  properties:
    virtualNetworkSubnetIdRef:
      name: my-cool-stack-subnet
  reclaimPolicy: Delete
  providerRef:
    name: my-cool-stack-azure-provider
```

Apply the `vnetrule.yaml`
```
kubectl apply -f vnetrule.yaml
```

Verify the `MySQLServerVirtualNetworkRule` has a `status.condition` of `Ready: true`.
```
kubectl get mysqlservervirtualnetworkrules.database.azure.crossplane.io -n workspace1 -o yaml
```
```
  status:
    conditions:
    - lastTransitionTime: "2020-04-30T02:18:22Z"
      reason: Resource is available for use
      status: "True"
      type: Ready
```

When cleaning up make sure to delete the `MySQLServerVirutalNetworkRule`
```
kubectl delete mysqlservervirtualnetworkrules.database.azure.crossplane.io my-cool-app-vnetrule
```

### AWS Networking DependencyViolation
After deleting the `AWSSample` you may notice that networking resources are stuck in the deleting state. This is typically a result of EKS leaking the `Load Balancer` and/or `Security Group` for a Kubernetes service after it's been deleted.

For example if you run:
```
kubectl get vpcs.network.aws.crossplane.io -o yaml
```

You'll get an error like the following:
```
  status:
    conditions:
    - lastTransitionTime: "2020-05-21T16:32:35Z"
      reason: Resource is being deleted
      status: "False"
      type: Ready
    - lastTransitionTime: "2020-05-21T16:32:36Z"
      message: "delete failed: failed to delete the VPC resource: DependencyViolation:
        The vpc 'vpc-095ed7a641825d3ef' has dependencies and cannot be deleted.\n\tstatus
        code: 400, request id: a0732f6e-d779-4224-b909-a7f948cc87eb"
      reason: Encountered an error during resource reconciliation
      status: "False"
      type: Synced
```

To workaround this, head to the AWS console and find the VPC with matching crossplane tags:
![aws-vpc-tags](aws-vpc-tags.png)

Then find and delete any associated `Load Balancers` that were orphaned by EKS for a Kubernetes service:
![aws-orphaned-lb](aws-orphaned-lb.png)

Then find and delete any associated `Security Groups` that were orphaned by EKS for a Kubernetes service:
![aws-orphaned-sg](aws-orphaned-sg.png)

Once this is done the VPC will be successfully deleted by Crossplane:
```
kubectl get vpcs.network.aws.crossplane.io -o yaml
```

Will show:
```
apiVersion: v1
items: []
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```



[install crossplane]: https://crossplane.io/docs/latest
[GCP Sample Stack]: https://github.com/crossplane/stack-gcp-sample
[AWS Sample Stack]: https://github.com/crossplane/stack-aws-sample
[Azure Sample Stack]: https://github.com/crossplane/stack-azure-sample
[Crossplane CLI]: https://github.com/crossplane/crossplane-cli
[Setup Environment]: #setup-environment
[Install Provider and Stack]: #install-provider-and-stack
[GCP Provider and Stack]: #gcp-provider-and-stack
[AWS Provider and Stack]: #aws-provider-and-stack
[Azure Provider and Stack]: #azure-provider-and-stack
[Install Application]: #install-application
[Cleanup]: #cleanup
[Debugging]: #debugging
[templating-controller]: https://github.com/crossplane/crossplane/blob/master/docs/contributing/experimental.md

[Debugging]: #Debugging
[MySQLInstance is Ready]: #MySQLInstance-is-Ready
[KubernetesCluster is Ready]: #KubernetesCluster-is-Ready
[Deployment and Service Debugging]: #Deployment-and-Service-Debugging
[Azure MySQLServerVirtualNetworkRule]: #Azure-MySQLServerVirtualNetworkRule
[AWS Networking DependencyViolation]: #AWS-Networking-DependencyViolation
[stack-aws-sample]: https://github.com/crossplane/stack-aws-sample/tree/master/kustomize/aws
