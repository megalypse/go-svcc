package commands

import (
	"encoding/json"
	"fmt"
	"os"

	cfg2 "github.com/megalypse/go-svc-cluster/cfg"
	"github.com/megalypse/go-svc-cluster/internal/domain/models"
	"github.com/spf13/cobra"
)

var CmdNewCluster = &cobra.Command{
	Use: "new",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := cfg2.GetConfig()
		appPath := cfg.Path
		fileName := fmt.Sprintf("%s.json", args[0])
		path := fmt.Sprintf("%s/%s", appPath, fileName)

		sampleCluster := &models.Cluster{
			Name: "",
			Nodes: []*models.Node{
				{
					Name:           "",
					HealthCheckURL: "",
					StartUpCommand: "",
					StartUpDir:     "",
					EnvVarPort:     "",
					OtherEnvVars: map[string]string{
						"EXAMPLE_ENV_VAR": "value",
					},
				},
			},
		}

		jsonBytes, err := json.Marshal(sampleCluster)
		if err != nil {
			fmt.Printf("Error marshaling JSON: %v\n", err)
			return
		}

		if _, err := os.Stat(appPath); os.IsNotExist(err) {
			err = os.MkdirAll(appPath, 0755)
			if err != nil {
				fmt.Printf("Error creating directory: %v\n", err)
				return
			}
		}

		// handle case where file already exists to prevent overwriting
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("File %s already exists. Aborting to prevent overwrite.\n", path)
			return
		}

		err = os.WriteFile(path, jsonBytes, 0644)
		if err != nil {
			fmt.Printf("Error writing file: %v\n", err)
			return
		}
	},
}
