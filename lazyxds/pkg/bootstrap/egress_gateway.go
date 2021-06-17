/*
 * // Copyright Aeraki Authors
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //     http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 */

package bootstrap

import (
	"context"
	"github.com/aeraki-framework/aeraki/lazyxds/cmd/lazyxds/app/config"
	"github.com/aeraki-framework/aeraki/lazyxds/pkg/utils"
	networking "istio.io/api/networking/v1alpha3"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"strings"
)

var lazyXdsEgressLabels = map[string]string{
	"app":   config.EgressName,
	"istio": "egressgateway",
}

// InitEgress ...
func InitEgress(
	ctx context.Context,
	name string,
	kubeClient *kubernetes.Clientset,
	istioClient *istioclient.Clientset,
	istiodAddress string,
	proxyImage string,
) error {
	if err := createEgressServiceAccount(ctx, kubeClient); err != nil {
		return err
	}
	if err := createEgressRole(ctx, kubeClient); err != nil {
		return err
	}
	if err := createEgressRoleBinding(ctx, kubeClient); err != nil {
		return err
	}
	if err := createAlsConfigMap(ctx, kubeClient); err != nil {
		return err
	}
	if err := CreateEgressEnvoyFilter(ctx, istioClient); err != nil {
		return err
	}
	if err := createEgressDeployment(ctx, name, kubeClient, istiodAddress, proxyImage); err != nil {
		return err
	}
	if err := createEgressService(ctx, kubeClient); err != nil {
		return err
	}
	if err := CreateEgressGateway(ctx, istioClient); err != nil {
		return err
	}
	return nil
}

func createEgressDeployment(
	ctx context.Context,
	name string,
	client *kubernetes.Clientset,
	istiodAddress string,
	proxyImage string,
) error {
	_, err := client.AppsV1().Deployments(config.IstioNamespace).Get(ctx, config.EgressName, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	optional := true
	replicas := int32(1)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.EgressName,
			Namespace: config.IstioNamespace,
			Labels:    lazyXdsEgressLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: lazyXdsEgressLabels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: lazyXdsEgressLabels,
					Annotations: map[string]string{
						"sidecar.istio.io/discoveryAddress": istiodAddress,
						"sidecar.istio.io/inject":           "false",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:  "istio-proxy",
						Image: proxyImage,
						Args: []string{
							"proxy", "router",
							"--domain", "$(POD_NAMESPACE).svc.cluster.local",
							"--proxyLogLevel=warning",
							"--proxyComponentLogLevel=misc:error",
							"--log_output_level=default:info",
							"--serviceCluster", "istio-egressgateway", //todo
						},
						Ports: []v1.ContainerPort{
							{ContainerPort: 8080},
							{ContainerPort: 15090, Name: "http-envoy-prom"},
						},
						Env: []v1.EnvVar{
							{Name: "ISTIO_BOOTSTRAP_OVERRIDE", Value: "/etc/istio/custom-bootstrap/custom_bootstrap.json"},
							{Name: "JWT_POLICY", Value: "first-party-jwt"},
							{Name: "PILOT_CERT_PROVIDER", Value: "istiod"},
							{Name: "CA_ADDR", Value: istiodAddress},
							{Name: "ISTIO_META_WORKLOAD_NAME", Value: config.EgressName},
							{Name: "ISTIO_META_OWNER", Value: "kubernetes://apis/apps/v1/namespaces/istio-system/deployments/" + config.EgressName},
							{Name: "ISTIO_META_MESH_ID", Value: "cluster.local"},
							{Name: "ISTIO_META_ROUTER_MODE", Value: "standard"},
							{Name: "ISTIO_META_CLUSTER_ID", Value: name}, // default should be Kubernetes, cluster of multiCluster should be different

							{Name: "NODE_NAME", ValueFrom: &v1.EnvVarSource{
								FieldRef: &v1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "spec.nodeName"},
							}},
							{Name: "POD_NAME", ValueFrom: &v1.EnvVarSource{
								FieldRef: &v1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.name"},
							}},
							{Name: "POD_NAMESPACE", ValueFrom: &v1.EnvVarSource{
								FieldRef: &v1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.namespace"},
							}},
							{Name: "INSTANCE_IP", ValueFrom: &v1.EnvVarSource{
								FieldRef: &v1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "status.podIP"},
							}},
							{Name: "HOST_IP", ValueFrom: &v1.EnvVarSource{
								FieldRef: &v1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "status.hostIP"},
							}},
							{Name: "SERVICE_ACCOUNT", ValueFrom: &v1.EnvVarSource{
								FieldRef: &v1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "spec.serviceAccountName"},
							}},
						},
						VolumeMounts: []v1.VolumeMount{
							{Name: "custom-bootstrap-volume", MountPath: "/etc/istio/custom-bootstrap"},
							{Name: "istio-envoy", MountPath: "/etc/istio/proxy"},
							{Name: "istiod-ca-cert", MountPath: "/var/run/secrets/istio"},
							{Name: "gatewaysdsudspath", MountPath: "/var/run/ingress_gateway"},
							{Name: "istio-data", MountPath: "/var/lib/istio/data"},
							{Name: "podinfo", MountPath: "/etc/istio/pod"},
							{Name: "egressgateway-certs", MountPath: "/etc/istio/egressgateway-certs", ReadOnly: true},
							{Name: "egressgateway-ca-certs", MountPath: "/etc/istio/egressgateway-ca-certs", ReadOnly: true},
						},
						ReadinessProbe: &v1.Probe{
							Handler: v1.Handler{
								HTTPGet: &v1.HTTPGetAction{
									Path:   "/healthz/ready",
									Port:   intstr.IntOrString{IntVal: 15021},
									Scheme: "HTTP",
								},
							},
							InitialDelaySeconds: 1,
							PeriodSeconds:       2,
							FailureThreshold:    30,
						},
						ImagePullPolicy: "IfNotPresent",
					}},
					Volumes: []v1.Volume{
						{
							Name: "custom-bootstrap-volume",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{Name: "lazyxds-als-bootstrap"},
								},
							},
						},
						{
							Name: "istiod-ca-cert",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{Name: "istio-ca-root-cert"},
								},
							},
						},
						{
							Name: "podinfo",
							VolumeSource: v1.VolumeSource{
								DownwardAPI: &v1.DownwardAPIVolumeSource{
									Items: []v1.DownwardAPIVolumeFile{
										{Path: "labels", FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.labels"}},
										{Path: "annotations", FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.annotations"}},
									},
								},
							},
						},
						{Name: "istio-envoy", VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}},
						{Name: "gatewaysdsudspath", VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}},
						{Name: "istio-data", VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}},

						{
							Name: "egressgateway-certs",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{SecretName: config.EgressName + "-certs", Optional: &optional},
							},
						},
						{
							Name: "egressgateway-ca-certs",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{SecretName: config.EgressName + "-ca-certs", Optional: &optional},
							},
						},
					},
					ServiceAccountName: config.EgressName + "-service-account",
				},
			},
		},
	}

	_, err = client.AppsV1().Deployments(config.IstioNamespace).Create(ctx, deploy, metav1.CreateOptions{
		FieldManager: config.LazyXdsManager,
	})
	return err
}

func createEgressService(ctx context.Context, client *kubernetes.Clientset) error {
	_, err := client.CoreV1().Services(config.IstioNamespace).Get(ctx, config.EgressName, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.EgressName,
			Namespace: config.IstioNamespace,
			Labels:    lazyXdsEgressLabels,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Name: "http", Port: config.EgressServicePort},
			},
			Selector: lazyXdsEgressLabels,
		},
	}

	_, err = client.CoreV1().Services(config.IstioNamespace).Create(ctx, service,
		metav1.CreateOptions{
			FieldManager: config.LazyXdsManager,
		},
	)
	return err
}

func createEgressServiceAccount(ctx context.Context, client *kubernetes.Clientset) error {
	name := config.EgressName + "-service-account"
	_, err := client.CoreV1().ServiceAccounts(config.IstioNamespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: config.IstioNamespace,
			Labels:    lazyXdsEgressLabels,
		},
	}

	_, err = client.CoreV1().ServiceAccounts(config.IstioNamespace).Create(ctx, sa,
		metav1.CreateOptions{
			FieldManager: config.LazyXdsManager,
		},
	)
	return err
}

func createEgressRole(ctx context.Context, client *kubernetes.Clientset) error {
	name := config.EgressName + "-sds"
	_, err := client.RbacV1().Roles(config.IstioNamespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: config.IstioNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}

	_, err = client.RbacV1().Roles(config.IstioNamespace).Create(ctx, role,
		metav1.CreateOptions{
			FieldManager: config.LazyXdsManager,
		},
	)
	return err
}

func createEgressRoleBinding(ctx context.Context, client *kubernetes.Clientset) error {
	name := config.EgressName + "-sds"
	_, err := client.RbacV1().RoleBindings(config.IstioNamespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: config.IstioNamespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: config.EgressName + "-service-account",
			},
		},
	}

	_, err = client.RbacV1().RoleBindings(config.IstioNamespace).Create(ctx, binding,
		metav1.CreateOptions{
			FieldManager: config.LazyXdsManager,
		},
	)
	return err
}

func createAlsConfigMap(ctx context.Context, client *kubernetes.Clientset) error {
	name := "lazyxds-als-bootstrap"
	_, err := client.CoreV1().ConfigMaps(config.IstioNamespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: config.IstioNamespace,
		},
		Data: map[string]string{
			"custom_bootstrap.json": alsBootstrapJSON(),
		},
	}

	_, err = client.CoreV1().ConfigMaps(config.IstioNamespace).Create(ctx, cm, metav1.CreateOptions{
		FieldManager: config.LazyXdsManager,
	})
	return err
}

// CreateEgressEnvoyFilter ...
func CreateEgressEnvoyFilter(ctx context.Context, client *istioclient.Clientset) error {
	name := "lazyxds-egress-als"

	_, err := client.NetworkingV1alpha3().EnvoyFilters(config.IstioNamespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	patch, err := utils.BuildPatchStruct(egressEnvoyFilterPatch())
	if err == nil {
		return nil
	}

	envoyFilter := &istio.EnvoyFilter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: config.IstioNamespace,
		},
		Spec: networking.EnvoyFilter{
			WorkloadSelector: &networking.WorkloadSelector{
				Labels: lazyXdsEgressLabels,
			},
			ConfigPatches: []*networking.EnvoyFilter_EnvoyConfigObjectPatch{
				{
					ApplyTo: networking.EnvoyFilter_NETWORK_FILTER,
					Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
						Context: networking.EnvoyFilter_GATEWAY,
						ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
							Listener: &networking.EnvoyFilter_ListenerMatch{
								FilterChain: &networking.EnvoyFilter_ListenerMatch_FilterChainMatch{
									Filter: &networking.EnvoyFilter_ListenerMatch_FilterMatch{
										Name: "envoy.filters.network.http_connection_manager",
									},
								},
							},
						},
					},
					Patch: &networking.EnvoyFilter_Patch{
						Operation: networking.EnvoyFilter_Patch_MERGE,
						Value:     patch,
					},
				},
			},
		},
	}

	_, err = client.NetworkingV1alpha3().EnvoyFilters(config.IstioNamespace).Create(ctx, envoyFilter, metav1.CreateOptions{})
	return err
}

// CreateEgressGateway ...
func CreateEgressGateway(ctx context.Context, client *istioclient.Clientset) error {
	name := config.EgressGatewayName

	_, err := client.NetworkingV1alpha3().Gateways(config.IstioNamespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	gw := &istio.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: config.IstioNamespace,
			Labels:    lazyXdsEgressLabels,
		},
		Spec: networking.Gateway{
			Servers: []*networking.Server{
				{
					Hosts: []string{"*"},
					Port: &networking.Port{
						Name:     "http",
						Number:   8080,
						Protocol: "HTTP",
					},
				},
			},
			Selector: lazyXdsEgressLabels,
		},
	}

	_, err = client.NetworkingV1alpha3().Gateways(config.IstioNamespace).Create(ctx, gw, metav1.CreateOptions{})
	return err
}

func egressEnvoyFilterPatch() string {
	return `{
  "typed_config": {
    "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
    "access_log": [
      {
        "name": "envoy.access_loggers.file",
        "typed_config": {
          "@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
          "path": "/dev/stdout",
          "log_format": {
            "text_format": "[%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% \"%UPSTREAM_TRANSPORT_FAILURE_REASON%\" %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\" \"%UPSTREAM_HOST%\" %UPSTREAM_CLUSTER% %UPSTREAM_LOCAL_ADDRESS% %DOWNSTREAM_LOCAL_ADDRESS% %DOWNSTREAM_REMOTE_ADDRESS% %REQUESTED_SERVER_NAME% %ROUTE_NAME%\n"
          }
        }
      },
      {
        "name": "envoy.access_loggers.http_grpc",
        "typed_config": {
          "@type": "type.googleapis.com/envoy.extensions.access_loggers.grpc.v3.HttpGrpcAccessLogConfig",
          "common_config": {
            "log_name": "http_envoy_accesslog",
            "transport_api_version": "V3",
            "grpc_service": {
              "envoy_grpc": {
                "cluster_name": "lazyxds-accesslog-service"
              }
            }
          }
        }
      }
    ]
  }
}`
}

func alsBootstrapJSON() string {
	s := `{
		"static_resources": {
			"clusters": [{
				"name": "lazyxds-accesslog-service",
				"type": "STRICT_DNS",
				"connect_timeout": "1s",
				"http2_protocol_options": {},
				"dns_lookup_family": "V4_ONLY",
				"load_assignment": {
					"cluster_name": "lazyxds-accesslog-service",
					"endpoints": [{
						"lb_endpoints": [{
							"endpoint": {
								"address": {
									"socket_address": {
										"address":    "lazyxds.istio-system",
										"port_value": 8080
									}
								}
							}
						}]
					}]
				},
				"respect_dns_ttl": true
			}]
		}
	}`

	return strings.ReplaceAll(strings.ReplaceAll(s, "\t", ""), "\n", "")
}
