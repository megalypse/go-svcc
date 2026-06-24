package impl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type ClusterInfo struct {
	ClusterId int
	NodeId    int
	Loading   bool
	Error     error
	Log       string
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
	clusterChan := make(chan *ClusterInfo, 64)
	commandEnv := commandEnvironment()
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

	setNodeLog := func(nodeId int, line string) {
		select {
		case clusterChan <- &ClusterInfo{
			ClusterId: clusterId,
			NodeId:    nodeId,
			Log:       line,
		}:
		case <-ctx.Done():
		}
	}

	streamLogs := func(nodeId int, reader io.Reader) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			setNodeLog(nodeId, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			setNodeLog(nodeId, fmt.Sprintf("log read error: %v", err))
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
				cmd.Env = commandEnv
				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

				stdout, err := cmd.StdoutPipe()
				if err != nil {
					return err
				}

				stderr, err := cmd.StderrPipe()
				if err != nil {
					return err
				}

				go func() {
					if err := cmd.Start(); err != nil {
						setNodeError(i, err)
						return
					}

					go streamLogs(i, stdout)
					go streamLogs(i, stderr)

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

func commandEnvironment() []string {
	env := os.Environ()
	cleanPath := cleanPath(os.Getenv("PATH"), "go")
	if cleanPath == "" {
		return env
	}

	pathSet := false
	for i, item := range env {
		if strings.HasPrefix(item, "PATH=") {
			env[i] = "PATH=" + cleanPath
			pathSet = true
			break
		}
	}

	if !pathSet {
		env = append(env, "PATH="+cleanPath)
	}

	return env
}

func cleanPath(pathValue string, commandNames ...string) string {
	if pathValue == "" {
		return pathValue
	}

	var cleanDirs []string
	for _, dir := range filepath.SplitList(pathValue) {
		if shadowsCommand(dir, commandNames...) {
			continue
		}
		cleanDirs = append(cleanDirs, dir)
	}

	return strings.Join(cleanDirs, string(os.PathListSeparator))
}

func shadowsCommand(dir string, commandNames ...string) bool {
	for _, commandName := range commandNames {
		info, err := os.Stat(filepath.Join(dir, commandName))
		if err != nil {
			continue
		}
		if info.IsDir() || info.Mode().Perm()&0111 == 0 {
			return true
		}
	}

	return false
}
