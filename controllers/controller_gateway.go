package controllers

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	seaweedv1 "github.com/seaweedfs/seaweedfs-operator/api/v1"
	label "github.com/seaweedfs/seaweedfs-operator/controllers/label"
)

const (
	GATEWAY_ROOT_USER     = "MINIO_ROOT_USER"
	GATEWAY_ROOT_PASSWORD = "MINIO_ROOT_PASSWORD"
)

func (r *SeaweedReconciler) ensureS3Gateway(seaweedCR *seaweedv1.Seaweed) (done bool, result ctrl.Result, err error) {
	_ = context.Background()
	_ = r.Log.WithValues("seaweed", seaweedCR.Name)

	if done, result, err = r.ensureGatewayService(seaweedCR); done {
		return
	}

	if done, result, err = r.ensureGatewaySecret(seaweedCR); done {
		return
	}

	if done, result, err = r.ensureGatewayDeployment(seaweedCR); done {
		return
	}

	return
}

func (r *SeaweedReconciler) ensureGatewayService(seaweedCR *seaweedv1.Seaweed) (bool, ctrl.Result, error) {

	log := r.Log.WithValues("sw-s3-gateway-service", seaweedCR.Name)

	gatewayService := r.createGatewayService(seaweedCR)
	if err := controllerutil.SetControllerReference(seaweedCR, gatewayService, r.Scheme); err != nil {
		return ReconcileResult(err)
	}
	_, err := r.CreateOrUpdateService(gatewayService)

	log.Info("ensure s3 gateway service " + gatewayService.Name)

	return ReconcileResult(err)
}

func (r *SeaweedReconciler) ensureGatewaySecret(seaweedCR *seaweedv1.Seaweed) (bool, ctrl.Result, error) {
	log := r.Log.WithValues("sw-s3-gateway-secret", seaweedCR.Name)

	gatewaySecret := r.createGatewaySecret(seaweedCR)
	if err := controllerutil.SetControllerReference(seaweedCR, gatewaySecret, r.Scheme); err != nil {
		return ReconcileResult(err)
	}
	_, err := r.CreateOrUpdateSecret(gatewaySecret)

	log.Info("Get s3 gateway Secret " + gatewaySecret.Name)
	return ReconcileResult(err)
}

func (r *SeaweedReconciler) ensureGatewayDeployment(seaweedCR *seaweedv1.Seaweed) (bool, ctrl.Result, error) {
	log := r.Log.WithValues("sw-s3-gateway-deployment", seaweedCR.Name)

	gatewayDeployment := r.createGatewayDeployment(seaweedCR)
	if err := controllerutil.SetControllerReference(seaweedCR, gatewayDeployment, r.Scheme); err != nil {
		return ReconcileResult(err)
	}
	_, err := r.CreateOrUpdateDeployment(gatewayDeployment)

	log.Info("ensure s3 gateway deployment " + gatewayDeployment.Name)

	return ReconcileResult(err)
}

func labelsForGateway(name string) map[string]string {
	return map[string]string{
		label.ManagedByLabelKey: "seaweedfs-operator",
		label.NameLabelKey:      "seaweedfs",
		label.ComponentLabelKey: "s3-gateway",
		label.InstanceLabelKey:  name,
	}
}
