package impl

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type ClusterInfo struct {
	ClusterId int
	NodeId    int
	Loading   bool
	Error     error
}

func NewStartClusterService() *StartClusterService {
	return &StartClusterService{
		runningClusters: []int{},
	}
}

type StartClusterService struct {
	runningClusters []int
}

func (s *StartClusterService) StartCluster(ctx context.Context, clusterId int) <-chan *ClusterInfo {
	clusters, _ := GetClusters()
	cluster := clusters[clusterId]
	clusterChan := make(chan *ClusterInfo, 1)
	setNodeLoading := func(nodeId int, loading bool) {
		select {
		case clusterChan <- &ClusterInfo{
			ClusterId: clusterId,
			NodeId:    nodeId,
			Loading:   loading,
		}:
		case <-ctx.Done():
		}
	}

	setNodeError := func(nodeId int, err error) {
		select {
		case clusterChan <- &ClusterInfo{
			ClusterId: clusterId,
			NodeId:    nodeId,
			Error:     err,
		}:
		case <-ctx.Done():
		}
	}

	go func(clusterChan chan<- *ClusterInfo, clusterId int) {
		for i, node := range cluster.Nodes {
			if err := func() error {
				setNodeLoading(i, true)
				envVarsBuilder := strings.Builder{}
				for k, v := range node.OtherEnvVars {
					envVarsBuilder.WriteString(fmt.Sprintf("export %s=%s;", k, v))
				}

				cmd := exec.Command("sh", "-c", fmt.Sprintf(`%scd %s && %s`, envVarsBuilder.String(), node.StartUpDir, node.StartUpCommand))
				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

				go func() {
					if err := cmd.Start(); err != nil {
						setNodeError(i, err)
						return
					}

					done := make(chan error, 1)
					go func() {
						done <- cmd.Wait()
					}()

					select {
					case <-ctx.Done():
						_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
						<-done
					case err := <-done:
						if err != nil {
							setNodeError(i, err)
						}
					}
				}()

				go func() {
					attempts := node.PollAttempts
					pollTicker := time.NewTicker(time.Duration(node.RetryInterval) * time.Second)
					defer pollTicker.Stop()

					for {
						select {
						case <-ctx.Done():
							setNodeError(i, fmt.Errorf("health check polling cancelled for node %d", i))
							return
						case <-pollTicker.C:
							if attempts <= 0 {
								setNodeError(i, fmt.Errorf("health check failed for node %d after max attempts", i))
								return
							}

							attempts--
							req, err := http.NewRequestWithContext(ctx, http.MethodGet, node.HealthCheckURL, nil)
							if err != nil {
								setNodeError(i, err)
								return
							}

							get, err := http.DefaultClient.Do(req)
							if err != nil || get.StatusCode != http.StatusOK {
								if get != nil {
									get.Body.Close()
								}
								setNodeLoading(i, true)
								continue
							}
							get.Body.Close()

							setNodeLoading(i, false)
							attempts = node.PollAttempts
						}
					}
				}()

				return nil
			}(); err != nil {
				setNodeError(i, err)
				return
			}
		}
	}(clusterChan, clusterId)

	return clusterChan
}
