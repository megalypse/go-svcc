package factory

import (
	"sync"

	"github.com/megalypse/go-svc-cluster/internal/domain/impl"
)

var GetServiceStartCluster = sync.OnceValue(getServiceStartCluster)

func getServiceStartCluster() *impl.StartClusterService {
	return impl.NewStartClusterService()
}
