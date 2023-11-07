package scaling

import (
	"strings"

	"github.com/test-network-function/cnf-certification-test/pkg/configuration"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiv1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	scalingv1 "k8s.io/api/autoscaling/v1"
)

func GetResourceHPA(hpaList []*scalingv1.HorizontalPodAutoscaler, name, namespace, kind string) *scalingv1.HorizontalPodAutoscaler {
	for _, hpa := range hpaList {
		if hpa.Spec.ScaleTargetRef.Kind == kind && hpa.Spec.ScaleTargetRef.Name == name && hpa.Namespace == namespace {
			return hpa
		}
	}
	return nil
}
func IsManaged(podSetName string, managedPodSet []configuration.ManagedDeploymentsStatefulsets) bool {
	for _, ps := range managedPodSet {
		if ps.Name == podSetName {
			return true
		}
	}
	return false
}

func CheckOwnerReference(ownerReference []apiv1.OwnerReference, crdFilter []configuration.CrdFilter, crds []*apiextv1.CustomResourceDefinition) bool {
	// Create a map to store the scalable status of each CRD.
	crdScalableMap := make(map[string]bool)
	for _, crd := range crds {
		for _, crdF := range crdFilter {
			if strings.HasSuffix(crd.Name, crdF.NameSuffix) {
				crdScalableMap[crd.Spec.Names.Kind] = crdF.Scalable
				break
			}
		}
	}

	// Iterate over the owner references and check if any of them are scalable CRDs.
	for _, owner := range ownerReference {
		if crdScalableMap[owner.Kind] {
			return true
		}
	}

	return false
}
