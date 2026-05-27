package impl

import (
	"encoding/json"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/megalypse/go-svc-cluster/cfg"
	"github.com/megalypse/go-svc-cluster/internal/domain/models"
)

var GetClusters = sync.OnceValues(loadClusters)

func loadClusters() ([]*models.Cluster, error) {
	appCfg, err := cfg.GetConfig()
	if err != nil {
		return nil, err
	}

	clusterConfigPath := appCfg.Path
	entries, err := os.ReadDir(clusterConfigPath)
	if err != nil {
		return nil, err
	}

	var clusters []*models.Cluster
	for _, dir := range entries {
		if !dir.IsDir() && strings.Contains(dir.Name(), ".json") {
			fullPath := path.Join(clusterConfigPath, dir.Name())
			configBytes, err := os.ReadFile(fullPath)
			if err != nil {
				return nil, err
			}

			cluster := &models.Cluster{}
			err = json.Unmarshal(configBytes, cluster)
			if err != nil {
				return nil, err
			}

			clusters = append(clusters, cluster)
		}
	}

	return clusters, nil
}
