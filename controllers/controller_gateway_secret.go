package controllers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	seaweedv1 "github.com/seaweedfs/seaweedfs-operator/api/v1"
)

const (
	secretNameTemplate = "%s-s3-admin"
)

func (r *SeaweedReconciler) createGatewaySecret(m *seaweedv1.Seaweed) *corev1.Secret {
	labels := labelsForFiler(m.Name)

	dep := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getGatewaySecretName(m),
			Namespace: m.Namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			GATEWAY_ROOT_USER:     []byte(m.Spec.Gateway.RootUser),
			GATEWAY_ROOT_PASSWORD: []byte(m.Spec.Gateway.RootPassword),
		},
	}
	return dep
}

func (r *SeaweedReconciler) getGatewaySecretName(m *seaweedv1.Seaweed) string {
	return fmt.Sprintf(secretNameTemplate, m.Name)
}

func (r *SeaweedReconciler) getGatewaySecretRefEnv(m *seaweedv1.Seaweed) (result []corev1.EnvVar) {
	result = append(result,
		[]corev1.EnvVar{
			{
				Name: GATEWAY_ROOT_USER,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: r.getGatewaySecretName(m),
						},
						Key: GATEWAY_ROOT_USER,
					},
				},
			},
			{
				Name: GATEWAY_ROOT_PASSWORD,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: r.getGatewaySecretName(m),
						},
						Key: GATEWAY_ROOT_PASSWORD,
					},
				},
			},
		}...,
	)
	return
}
