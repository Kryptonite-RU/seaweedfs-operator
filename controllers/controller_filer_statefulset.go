package controllers

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	seaweedv1 "github.com/seaweedfs/seaweedfs-operator/api/v1"
)

func buildFilerStartupScript(m *seaweedv1.Seaweed) string {
	commands := []string{"weed", "-logtostderr=true", "filer"}
	commands = append(commands, fmt.Sprintf("-port=%d", seaweedv1.FilerHTTPPort))
	commands = append(commands, fmt.Sprintf("-ip=$(POD_NAME).%s-filer-peer.%s", m.Name, m.Namespace))
	commands = append(commands, fmt.Sprintf("-master=%s", getMasterPeersString(m)))
	if *m.Spec.Filer.S3 {
		commands = append(commands, "-s3")
	}

	return strings.Join(commands, " ")
}

func (r *SeaweedReconciler) createFilerStatefulSet(m *seaweedv1.Seaweed) *appsv1.StatefulSet {
	labels := labelsForFiler(m.Name)
	replicas := int32(m.Spec.Filer.Replicas)
	rollingUpdatePartition := int32(0)
	enableServiceLinks := false

	requestCPU := m.Spec.Filer.Requests[corev1.ResourceCPU]
	requestMemory := m.Spec.Filer.Requests[corev1.ResourceMemory]
	limitCPU := m.Spec.Filer.Limits[corev1.ResourceCPU]
	limitMemory := m.Spec.Filer.Limits[corev1.ResourceMemory]

	resources := corev1.ResourceRequirements{}

	if !limitCPU.IsZero() || !limitMemory.IsZero() {
		resources.Limits = corev1.ResourceList{}

		if !limitCPU.IsZero() {
			resources.Limits[corev1.ResourceCPU] = limitCPU
		}

		if !limitMemory.IsZero() {
			resources.Limits[corev1.ResourceMemory] = limitMemory
		}
	}

	if !requestCPU.IsZero() || !requestMemory.IsZero() {
		resources.Requests = corev1.ResourceList{}

		if !requestCPU.IsZero() {
			resources.Requests[corev1.ResourceCPU] = requestCPU
		}

		if !requestMemory.IsZero() {
			resources.Requests[corev1.ResourceMemory] = requestMemory
		}
	}

	filerPodSpec := m.BaseFilerSpec().BuildPodSpec()
	filerPodSpec.Volumes = []corev1.Volume{
		{
			Name: "filer-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: m.Name + "-filer",
					},
				},
			},
		},
	}
	filerPodSpec.EnableServiceLinks = &enableServiceLinks
	filerPodSpec.Containers = []corev1.Container{{
		Name:            "filer",
		Image:           m.Spec.Image,
		ImagePullPolicy: m.BaseFilerSpec().ImagePullPolicy(),
		Env:             append(m.BaseFilerSpec().Env(), kubernetesEnvVars...),
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "filer-config",
				ReadOnly:  true,
				MountPath: "/etc/seaweedfs",
			},
		},
		Command: []string{
			"/bin/sh",
			"-ec",
			buildFilerStartupScript(m),
		},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: seaweedv1.FilerHTTPPort,
				Name:          "filer-http",
			},
			{
				ContainerPort: seaweedv1.FilerGRPCPort,
				Name:          "filer-grpc",
			},
			{
				ContainerPort: seaweedv1.FilerS3Port,
				Name:          "filer-s3",
			},
		},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/",
					Port:   intstr.FromInt(seaweedv1.FilerHTTPPort),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 10,
			TimeoutSeconds:      3,
			PeriodSeconds:       15,
			SuccessThreshold:    1,
			FailureThreshold:    100,
		},
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/",
					Port:   intstr.FromInt(seaweedv1.FilerHTTPPort),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 20,
			TimeoutSeconds:      3,
			PeriodSeconds:       30,
			SuccessThreshold:    1,
			FailureThreshold:    6,
		},
		Resources: resources,
	}}

	dep := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-filer",
			Namespace: m.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName:         m.Name + "-filer-peer",
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Replicas:            &replicas,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
					Partition: &rollingUpdatePartition,
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: filerPodSpec,
			},
		},
	}
	return dep
}
