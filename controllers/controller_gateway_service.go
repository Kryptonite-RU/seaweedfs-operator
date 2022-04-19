package controllers

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	seaweedv1 "github.com/seaweedfs/seaweedfs-operator/api/v1"
)

func (r *SeaweedReconciler) createGatewayService(m *seaweedv1.Seaweed) *corev1.Service {
	labels := labelsForGateway(m.Name)

	ports := []corev1.ServicePort{
		{
			Name:       "gateway-http",
			Protocol:   corev1.Protocol("TCP"),
			Port:       seaweedv1.GatewayConsolePort,
			TargetPort: intstr.FromInt(seaweedv1.GatewayConsolePort),
		},
		{
			Name:       "gateway-s3",
			Protocol:   corev1.Protocol("TCP"),
			Port:       seaweedv1.GatewayPort,
			TargetPort: intstr.FromInt(seaweedv1.GatewayPort),
		},
	}

	dep := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-s3",
			Namespace: m.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Ports:    ports,
			Selector: labels,
		},
	}

	if m.Spec.Gateway.Service != nil {
		svcSpec := m.Spec.Gateway.Service
		dep.Annotations = copyAnnotations(svcSpec.Annotations)

		if svcSpec.Type != "" {
			dep.Spec.Type = svcSpec.Type
		}

		if svcSpec.ClusterIP != nil {
			dep.Spec.ClusterIP = *svcSpec.ClusterIP
		}

		if svcSpec.LoadBalancerIP != nil {
			dep.Spec.LoadBalancerIP = *svcSpec.LoadBalancerIP
		}
	}

	return dep
}
