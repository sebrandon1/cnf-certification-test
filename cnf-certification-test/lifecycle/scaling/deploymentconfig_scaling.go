// Copyright (C) 2020-2022 Red Hat, Inc.
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, write to the Free Software Foundation, Inc.,
// 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.

//nolint:dupl
package scaling

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/test-network-function/cnf-certification-test/cnf-certification-test/lifecycle/podsets"
	"github.com/test-network-function/cnf-certification-test/internal/clientsholder"
	"github.com/test-network-function/cnf-certification-test/pkg/provider"

	v1autoscaling "k8s.io/api/autoscaling/v1"

	v1machinery "k8s.io/apimachinery/pkg/apis/meta/v1"
	retry "k8s.io/client-go/util/retry"

	ocptypesv1 "github.com/openshift/api/apps/v1"
	hps "k8s.io/client-go/kubernetes/typed/autoscaling/v1"
)

func TestScaleDeploymentConfig(deploymentConfig *ocptypesv1.DeploymentConfig, timeout time.Duration) bool {
	clients := clientsholder.GetClientsHolder()
	logrus.Trace("scale deployment not using HPA ", deploymentConfig.Namespace, ":", deploymentConfig.Name)
	replicas := deploymentConfig.Spec.Replicas

	if replicas <= 1 {
		// scale up
		replicas++
		if !scaleDeploymentConfigHelper(clients, deploymentConfig, replicas, timeout, true) {
			logrus.Error("can't scale deployment =", deploymentConfig.Namespace, ":", deploymentConfig.Name)
			return false
		}
		// scale down
		replicas--
		if !scaleDeploymentConfigHelper(clients, deploymentConfig, replicas, timeout, false) {
			logrus.Error("can't scale deployment =", deploymentConfig.Namespace, ":", deploymentConfig.Name)
			return false
		}
	} else {
		// scale down
		replicas--
		if !scaleDeploymentConfigHelper(clients, deploymentConfig, replicas, timeout, false) {
			logrus.Error("can't scale deployment =", deploymentConfig.Namespace, ":", deploymentConfig.Name)
			return false
		} // scale up
		replicas++
		if !scaleDeploymentConfigHelper(clients, deploymentConfig, replicas, timeout, true) {
			logrus.Error("can't scale deployment =", deploymentConfig.Namespace, ":", deploymentConfig.Name)
			return false
		}
	}
	return true
}

func scaleDeploymentConfigHelper(clients *clientsholder.ClientsHolder, deploymentConfig *ocptypesv1.DeploymentConfig, replicas int32, timeout time.Duration, up bool) bool {
	if up {
		logrus.Trace("scale UP deployment to ", replicas, " replicas ")
	} else {
		logrus.Trace("scale DOWN deployment to ", replicas, " replicas ")
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Retrieve the latest version of DeploymentConfig before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
		dp, err := clients.OcpAppsClient.DeploymentConfigs(deploymentConfig.Namespace).Get(context.TODO(), deploymentConfig.Name, v1machinery.GetOptions{})
		if err != nil {
			logrus.Error("failed to get latest version of deployment ", deploymentConfig.Namespace, ":", deploymentConfig.Name)
			return err
		}
		dp.Spec.Replicas = replicas
		_, err = clients.OcpAppsClient.DeploymentConfigs(deploymentConfig.Namespace).Update(context.TODO(), dp, v1machinery.UpdateOptions{})
		if err != nil {
			logrus.Error("can't update deployment ", deploymentConfig.Namespace, ":", deploymentConfig.Name)
			return err
		}
		if !podsets.WaitForDeploymentConfigSetReady(deploymentConfig.Namespace, deploymentConfig.Name, timeout) {
			logrus.Error("can't update deployment ", deploymentConfig.Namespace, ":", deploymentConfig.Name)
			return errors.New("can't update deployment")
		}
		return nil
	})
	if retryErr != nil {
		logrus.Error("can't scale deployment ", deploymentConfig.Namespace, ":", deploymentConfig.Name, " error=", retryErr)
		return false
	}
	return true
}

func TestScaleHpaDeploymentConfig(deploymentConfig *provider.DeploymentConfig, hpa *v1autoscaling.HorizontalPodAutoscaler, timeout time.Duration) bool {
	clients := clientsholder.GetClientsHolder()
	hpscaler := clients.K8sClient.AutoscalingV1().HorizontalPodAutoscalers(deploymentConfig.Namespace)
	var min int32
	if hpa.Spec.MinReplicas != nil {
		min = *hpa.Spec.MinReplicas
	} else {
		min = 1
	}
	replicas := deploymentConfig.Spec.Replicas
	max := hpa.Spec.MaxReplicas
	if replicas <= 1 {
		// scale up
		replicas++
		logrus.Trace("scale UP HPA ", deploymentConfig.Namespace, ":", hpa.Name, "To min=", replicas, " max=", replicas)
		pass := scaleHpaDeploymenConfigtHelper(hpscaler, hpa.Name, deploymentConfig.Name, deploymentConfig.Namespace, replicas, replicas, timeout)
		if !pass {
			return false
		}
		// scale down
		replicas--
		logrus.Trace("scale DOWN HPA ", deploymentConfig.Namespace, ":", hpa.Name, "To min=", replicas, " max=", replicas)
		pass = scaleHpaDeploymenConfigtHelper(hpscaler, hpa.Name, deploymentConfig.Name, deploymentConfig.Namespace, min, max, timeout)
		if !pass {
			return false
		}
	} else {
		// scale down
		replicas--
		logrus.Trace("scale DOWN HPA ", deploymentConfig.Namespace, ":", hpa.Name, "To min=", replicas, " max=", replicas)
		pass := scaleHpaDeploymenConfigtHelper(hpscaler, hpa.Name, deploymentConfig.Name, deploymentConfig.Namespace, replicas, replicas, timeout)
		if !pass {
			return false
		}
		// scale up
		replicas++
		logrus.Trace("scale UP HPA ", deploymentConfig.Namespace, ":", hpa.Name, "To min=", replicas, " max=", replicas)
		pass = scaleHpaDeploymenConfigtHelper(hpscaler, hpa.Name, deploymentConfig.Name, deploymentConfig.Namespace, replicas, replicas, timeout)
		if !pass {
			return false
		}
	}
	// back the min and the max value of the hpa
	logrus.Trace("back HPA ", deploymentConfig.Namespace, ":", hpa.Name, "To min=", min, " max=", max)
	pass := scaleHpaDeploymenConfigtHelper(hpscaler, hpa.Name, deploymentConfig.Name, deploymentConfig.Namespace, min, max, timeout)
	return pass
}

func scaleHpaDeploymenConfigtHelper(hpscaler hps.HorizontalPodAutoscalerInterface, hpaName, deploymentName, namespace string, min, max int32, timeout time.Duration) bool {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		hpa, err := hpscaler.Get(context.TODO(), hpaName, v1machinery.GetOptions{})
		if err != nil {
			logrus.Error("can't Update autoscaler to scale ", namespace, ":", deploymentName, " error=", err)
			return err
		}
		hpa.Spec.MinReplicas = &min
		hpa.Spec.MaxReplicas = max
		_, err = hpscaler.Update(context.TODO(), hpa, v1machinery.UpdateOptions{})
		if err != nil {
			logrus.Error("can't Update autoscaler to scale ", namespace, ":", deploymentName, " error=", err)
			return err
		}
		if !podsets.WaitForDeploymentConfigSetReady(namespace, deploymentName, timeout) {
			logrus.Error("deploymentConfig not ready after scale operation ", namespace, ":", deploymentName)
		}
		return nil
	})
	if retryErr != nil {
		logrus.Error("can't scale hpa ", namespace, ":", hpaName, " error=", retryErr)
		return false
	}
	return true
}
