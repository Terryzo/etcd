// Copyright 2016 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration

import (
	"math/rand"
	"testing"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/integration"
	"github.com/coreos/etcd/pkg/testutil"
	"golang.org/x/net/context"
)

// TestDialSetEndpoints ensures SetEndpoints can replace unavailable endpoints with available ones.
func TestDialSetEndpointsBeforeFail(t *testing.T) {
	testDialSetEndpoints(t, true)
}

func TestDialSetEndpointsAfterFail(t *testing.T) {
	testDialSetEndpoints(t, false)
}

// testDialSetEndpoints ensures SetEndpoints can replace unavailable endpoints with available ones.
func testDialSetEndpoints(t *testing.T, setBefore bool) {
	defer testutil.AfterTest(t)
	clus := integration.NewClusterV3(t, &integration.ClusterConfig{Size: 3})
	defer clus.Terminate(t)

	// get endpoint list
	eps := make([]string, 3)
	for i := range eps {
		eps[i] = clus.Members[i].GRPCAddr()
	}
	toKill := rand.Intn(len(eps))

	cfg := clientv3.Config{Endpoints: []string{eps[toKill]}, DialTimeout: 1 * time.Second}
	cli, err := clientv3.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	if setBefore {
		cli.SetEndpoints(eps[toKill%3], eps[(toKill+1)%3])
	}
	// make a dead node
	clus.Members[toKill].Stop(t)
	clus.WaitLeader(t)

	if !setBefore {
		cli.SetEndpoints(eps[toKill%3], eps[(toKill+1)%3])
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	if _, err = cli.Get(ctx, "foo", clientv3.WithSerializable()); err != nil {
		t.Fatal(err)
	}
	cancel()
}

// TestSwitchSetEndpoints ensures SetEndpoints can switch one endpoint
// with a new one that doesn't include original endpoint.
func TestSwitchSetEndpoints(t *testing.T) {
	defer testutil.AfterTest(t)
	clus := integration.NewClusterV3(t, &integration.ClusterConfig{Size: 3})
	defer clus.Terminate(t)

	// get non partitioned members endpoints
	eps := []string{clus.Members[1].GRPCAddr(), clus.Members[2].GRPCAddr()}

	cli := clus.Client(0)
	clus.Members[0].InjectPartition(t, clus.Members[1:])

	cli.SetEndpoints(eps...)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := cli.Get(ctx, "foo"); err != nil {
		t.Fatal(err)
	}
}

func TestRejectOldCluster(t *testing.T) {
	defer testutil.AfterTest(t)
	// 2 endpoints to test multi-endpoint Status
	clus := integration.NewClusterV3(t, &integration.ClusterConfig{Size: 2})
	defer clus.Terminate(t)

	cfg := clientv3.Config{
		Endpoints:        []string{clus.Members[0].GRPCAddr(), clus.Members[1].GRPCAddr()},
		DialTimeout:      5 * time.Second,
		RejectOldCluster: true,
	}
	cli, err := clientv3.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cli.Close()
}
