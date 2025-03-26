package energyaware

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

func getInternalIP(node *v1.Node) (string, error) {
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			return addr.Address, nil
		}
	}
	return "", fmt.Errorf("[EnergyAware] no internal IP found for node %s", node.Name)
}
