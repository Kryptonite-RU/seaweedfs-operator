package controllers

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	seaweedv1 "github.com/seaweedfs/seaweedfs-operator/api/v1"
)

func buildGatewayStartupArgs(m *seaweedv1.Seaweed) []string {
	args := []string{
		"gateway",
		"weed",
		"--console-address",
		fmt.Sprintf(":%d", seaweedv1.GatewayConsolePort),
		"--address",
		fmt.Sprintf(":%d", seaweedv1.GatewayPort),
		fmt.Sprintf("%s-filer.%s:%d", m.Name, m.Namespace, seaweedv1.FilerHTTPPort),
		fmt.Sprintf("%s-master.%s:%d", m.Name, m.Namespace, seaweedv1.MasterHTTPPort),
	}

	return args
}

func (r *SeaweedReconciler) createGatewayDeployment(m *seaweedv1.Seaweed) *appsv1.Deployment {
	labels := labelsForGateway(m.Name)
	replicas := int32(m.Spec.Gateway.Replicas)

	envs := append(m.BaseGatewaySpec().Env(), kubernetesEnvVars...)
	envs = append(envs, r.getGatewaySecretRefEnv(m)...)

	gatewayPodSpec := m.BaseGatewaySpec().BuildPodSpec()
	gatewayPodSpec.Containers = []corev1.Container{{
		Name:            "s3-gateway",
		Image:           m.Spec.Gateway.Image,
		ImagePullPolicy: m.BaseGatewaySpec().ImagePullPolicy(),
		Env:             envs,

		Command: []string{
			"/opt/bin/minio",
		},

		Args: buildGatewayStartupArgs(m),

		Ports: []corev1.ContainerPort{
			{
				ContainerPort: seaweedv1.GatewayConsolePort,
				Name:          "gateway-http",
			},
			{
				ContainerPort: seaweedv1.GatewayPort,
				Name:          "gateway-s3",
			},
		},

		// TODO: add ReadinessProbe and LivenessProbe
	}}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-s3-gateway",
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: gatewayPodSpec,
			},
		},
	}

	return dep
}
