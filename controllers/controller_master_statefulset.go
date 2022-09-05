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

func buildMasterStartupScript(m *seaweedv1.Seaweed) string {
	command := []string{"weed", "-logtostderr=true", "master"}
	spec := m.Spec.Master
	if spec.VolumePreallocate != nil && *spec.VolumePreallocate {
		command = append(command, "-volumePreallocate")
	}

	if spec.VolumeSizeLimitMB != nil {
		command = append(command, fmt.Sprintf("-volumeSizeLimitMB=%d", *spec.VolumeSizeLimitMB))
	}

	if spec.GarbageThreshold != nil {
		command = append(command, fmt.Sprintf("-garbageThreshold=%s", *spec.GarbageThreshold))
	}

	if spec.PulseSeconds != nil {
		command = append(command, fmt.Sprintf("-pulseSeconds=%d", *spec.PulseSeconds))
	}

	if spec.DefaultReplication != nil {
		command = append(command, fmt.Sprintf("-defaultReplication=%s", *spec.DefaultReplication))
	}

	command = append(command, fmt.Sprintf("-ip=$(POD_NAME).%s-master-peer.%s", m.Name, m.Namespace))
	command = append(command, fmt.Sprintf("-peers=%s", getMasterPeersString(m)))
	command = append(command, fmt.Sprintf("-metricsPort=9999"))	
	return strings.Join(command, " ")
}

func (r *SeaweedReconciler) createMasterStatefulSet(m *seaweedv1.Seaweed) *appsv1.StatefulSet {
	labels := labelsForMaster(m.Name)
	replicas := m.Spec.Master.Replicas
	rollingUpdatePartition := int32(0)
	enableServiceLinks := false

	requestCPU := m.Spec.Master.Requests[corev1.ResourceCPU]
	requestMemory := m.Spec.Master.Requests[corev1.ResourceMemory]
	limitCPU := m.Spec.Master.Limits[corev1.ResourceCPU]
	limitMemory := m.Spec.Master.Limits[corev1.ResourceMemory]

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

	masterPodSpec := m.BaseMasterSpec().BuildPodSpec()
	masterPodSpec.Volumes = []corev1.Volume{
		{
			Name: "master-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: m.Name + "-master",
					},
				},
			},
		},
	}
	masterPodSpec.EnableServiceLinks = &enableServiceLinks
	masterPodSpec.Containers = []corev1.Container{{
		Name:            "master",
		Image:           m.Spec.Image,
		ImagePullPolicy: m.BaseMasterSpec().ImagePullPolicy(),
		Env:             append(m.BaseMasterSpec().Env(), kubernetesEnvVars...),
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "master-config",
				ReadOnly:  true,
				MountPath: "/etc/seaweedfs",
			},
		},
		Command: []string{
			"/bin/sh",
			"-ec",
			buildMasterStartupScript(m),
		},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: seaweedv1.MasterHTTPPort,
				Name:          "master-http",
			},
			{
				ContainerPort: seaweedv1.MasterGRPCPort,
				Name:          "master-grpc",
			},
		},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/cluster/status",
					Port:   intstr.FromInt(seaweedv1.MasterHTTPPort),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 5,
			TimeoutSeconds:      15,
			PeriodSeconds:       15,
			SuccessThreshold:    2,
			FailureThreshold:    100,
		},
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/cluster/status",
					Port:   intstr.FromInt(seaweedv1.MasterHTTPPort),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 15,
			TimeoutSeconds:      15,
			PeriodSeconds:       15,
			SuccessThreshold:    1,
			FailureThreshold:    6,
		},
		Resources: resources,
	}}

	dep := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-master",
			Namespace: m.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName:         m.Name + "-master-peer",
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
				Spec: masterPodSpec,
			},
		},
	}
	// Set master instance as the owner and controller
	// ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}
