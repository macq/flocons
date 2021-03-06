package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/samuel/go-zookeeper/zk"

	log "github.com/sirupsen/logrus"
	"github.com/t-mind/flocons/cluster"
	"github.com/t-mind/flocons/config"
	"github.com/t-mind/flocons/test/mock"
)

func createConfig(t *testing.T, number int) *config.Config {
	json_config := fmt.Sprintf(`{"namespace": "test", "node": {"name": "node-%d", "port": %d}, "storage": {"path": %q}}`, number, 5555+number, "/tmp")
	config, err := config.NewConfigFromJson([]byte(json_config))
	if err != nil {
		t.Errorf("Could not parse config %s: %s", json_config, err)
		t.FailNow()
	}
	return config
}

func TestMultipleTopologyClients(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	zookeeper := mock.NewZookeeper()
	defer zookeeper.Clear()
	numClients := 5
	clients := make([]cluster.TopologyClient, numClients)
	paths := make([]string, numClients)
	pathCreated := make([]bool, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = cluster.NewTopologyClientWithZookeperClientFactory(createConfig(t, i), zookeeper.GetFactory(), &mock.NullDispatcher{})
		defer clients[i].Close()
		paths[i] = fmt.Sprintf("/flocons/test/node-%d", i)
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 1000*time.Millisecond)
loop:
	for {
		select {
		case event, more := <-zookeeper.Events:
			if !more {
				break loop
			}
			fmt.Printf("Received event of type %s for path %s with state %s\n", event.Type, event.Path, event.State)
			if event.Type == zk.EventNodeCreated {
				allCreated := true
				for i, p := range paths {
					if event.Path == p {
						pathCreated[i] = true
					} else if pathCreated[i] == false {
						allCreated = false
					}
				}
				if allCreated {
					break loop
				}
			}
		case <-ctx.Done():
			break loop
		}
	}
	cancel()
	time.Sleep(10 * time.Millisecond) // Let nodes time to adjust
	for i, created := range pathCreated {
		if !created {
			t.Errorf("Client %d has not created its node %s", i, paths[i])
		}
	}
	for i, client := range clients {
		for j, _ := range clients {
			if i == j {
				continue
			}
			otherClientNodeName := fmt.Sprintf("node-%d", j)
			node, ok := client.Nodes()[otherClientNodeName]
			if !ok {
				t.Errorf("Client %d has not discovered client %d", i, j)
			} else if node.Name != otherClientNodeName {
				t.Errorf("Client %d info seems badly encoded", j)
			}
		}
	}

	clients[0].Close()

	closeDetected := false
	ctx = context.Background()
	ctx, cancel = context.WithTimeout(ctx, 10000*time.Millisecond)
loop2:
	for {
		select {
		case event, more := <-zookeeper.Events:
			if !more {
				break loop2
			}
			fmt.Printf("Received event of type %s for path %s with state %s\n", event.Type, event.Path, event.State)
			if event.Type == zk.EventSession && event.State == zk.StateDisconnected {
				closeDetected = true
				break loop2
			}
		case <-ctx.Done():
			break loop2
		}
	}
	cancel()
	time.Sleep(10 * time.Millisecond) // Let nodes time to adjust
	if !closeDetected {
		t.Error("Client close has not been detected")
	}
	if len(clients[0].Nodes()) > 0 {
		t.Error("Client 0 node list is not empty")
	}
	for i, client := range clients {
		if _, ok := client.Nodes()["node-0"]; ok && i != 0 {
			t.Errorf("Client %d has stil discovered client 0", i)
		}
	}
}
