// Copyright (C) 2022 Red Hat, Inc.
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

package provider

import (
	"fmt"

	ocptypesv1 "github.com/openshift/api/apps/v1"
	clientappsv1 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	"github.com/test-network-function/cnf-certification-test/pkg/autodiscover"
)

type DeploymentConfig struct {
	*ocptypesv1.DeploymentConfig
}

func (d *DeploymentConfig) ToString() string {
	return fmt.Sprintf("deploymentConfig: %s ns: %s",
		d.Name,
		d.Namespace,
	)
}

func (d *DeploymentConfig) IsDeploymentConfigReady() bool {
	notReady := true

	// Check the deployment's conditions for deploymentAvailable.
	for _, condition := range d.Status.Conditions {
		if condition.Type == ocptypesv1.DeploymentAvailable {
			notReady = false // Deployment is ready
			break
		}
	}

	// Find the number of expected replicas
	replicas := d.Spec.Replicas

	// If condition says that the deployment is not ready or replicas do not match totals specified in spec.replicas.
	if notReady ||
		d.Status.UnavailableReplicas != 0 || //
		d.Status.ReadyReplicas != replicas || // eg. 10 ready replicas == 10 total replicas
		d.Status.AvailableReplicas != replicas ||
		d.Status.UpdatedReplicas != replicas {
		return false
	}
	return true
}

func GetUpdatedDeploymentConfig(ac clientappsv1.AppsV1Interface, namespace, podName string) (*DeploymentConfig, error) {
	result, err := autodiscover.FindDeploymentConfigByNameByNamespace(ac, namespace, podName)
	return &DeploymentConfig{
		result,
	}, err
}
