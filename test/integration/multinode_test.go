// +build integration

/*
Copyright 2020 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestMultiNode(t *testing.T) {
	if NoneDriver() {
		t.Skip("none driver does not support multinode")
	}
	MaybeParallel(t)

	type validatorFunc func(context.Context, *testing.T, string)
	profile := UniqueProfileName("multinode")
	ctx, cancel := context.WithTimeout(context.Background(), Minutes(30))
	defer CleanupWithLogs(t, profile, cancel)

	t.Run("serial", func(t *testing.T) {
		tests := []struct {
			name      string
			validator validatorFunc
		}{
			{"FreshStart2Nodes", validateMultiNodeStart},
			{"AddNode", validateAddNodeToMultiNode},
			{"StopNode", validateStopRunningNode},
			{"StartAfterStop", validateStartNodeAfterStop},
			{"DeleteNode", validateDeleteNodeFromMultiNode},
		}
		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				tc.validator(ctx, t, profile)
			})
		}
	})
}

func validateMultiNodeStart(ctx context.Context, t *testing.T, profile string) {
	// Start a 2 node cluster with the --nodes param
	startArgs := append([]string{"start", "-p", profile, "--wait=true", "--nodes=2"}, StartArgs()...)
	rr, err := Run(t, exec.CommandContext(ctx, Target(), startArgs...))
	if err != nil {
		t.Fatalf("failed to start cluster. args %q : %v", rr.Command(), err)
	}

	// Make sure minikube status shows 2 nodes
	rr, err = Run(t, exec.CommandContext(ctx, Target(), "-p", profile, "status"))
	if err != nil {
		t.Fatalf("failed to run minikube status. args %q : %v", rr.Command(), err)
	}

	if strings.Count(rr.Stdout.String(), "host: Running") != 2 {
		t.Errorf("status says both hosts are not running: args %q: %v", rr.Command(), rr.Stdout.String())
	}

	if strings.Count(rr.Stdout.String(), "kubelet: Running") != 2 {
		t.Errorf("status says both kubelets are not running: args %q: %v", rr.Command(), rr.Stdout.String())
	}

}

func validateAddNodeToMultiNode(ctx context.Context, t *testing.T, profile string) {
	// Add a node to the current cluster
	addArgs := []string{"node", "add", "-p", profile, "-v", "3", "--alsologtostderr"}
	rr, err := Run(t, exec.CommandContext(ctx, Target(), addArgs...))
	if err != nil {
		t.Fatalf("failed to add node to current cluster. args %q : %v", rr.Command(), err)
	}

	// Make sure minikube status shows 3 nodes
	rr, err = Run(t, exec.CommandContext(ctx, Target(), "-p", profile, "status"))
	if err != nil {
		t.Fatalf("failed to run minikube status. args %q : %v", rr.Command(), err)
	}

	if strings.Count(rr.Stdout.String(), "host: Running") != 3 {
		t.Errorf("status says all hosts are not running: args %q: %v", rr.Command(), rr.Stdout.String())
	}

	if strings.Count(rr.Stdout.String(), "kubelet: Running") != 3 {
		t.Errorf("status says all kubelets are not running: args %q: %v", rr.Command(), rr.Stdout.String())
	}
}

func validateStopRunningNode(ctx context.Context, t *testing.T, profile string) {
	// Names are autogenerated using the node.Name() function
	name := "m03"

	// Run minikube node stop on that node
	rr, err := Run(t, exec.CommandContext(ctx, Target(), "-p", profile, "node", "stop", name))
	if err != nil {
		t.Errorf("node stop returned an error. args %q: %v", rr.Command(), err)
	}

	// Run status again to see the stopped host
	rr, err = Run(t, exec.CommandContext(ctx, Target(), "-p", profile, "status"))
	// Exit code 7 means one host is stopped, which we are expecting
	if err != nil && rr.ExitCode != 7 {
		t.Fatalf("failed to run minikube status. args %q : %v", rr.Command(), err)
	}

	// Make sure minikube status shows 2 running nodes and 1 stopped one
	rr, err = Run(t, exec.CommandContext(ctx, Target(), "-p", profile, "status"))
	if err != nil && rr.ExitCode != 7 {
		t.Fatalf("failed to run minikube status. args %q : %v", rr.Command(), err)
	}

	if strings.Count(rr.Stdout.String(), "kubelet: Running") != 2 {
		t.Errorf("incorrect number of running kubelets: args %q: %v", rr.Command(), rr.Stdout.String())
	}

	if strings.Count(rr.Stdout.String(), "host: Stopped") != 1 {
		t.Errorf("incorrect number of stopped hosts: args %q: %v", rr.Command(), rr.Stdout.String())
	}

	if strings.Count(rr.Stdout.String(), "kubelet: Stopped") != 1 {
		t.Errorf("incorrect number of stopped kubelets: args %q: %v", rr.Command(), rr.Stdout.String())
	}
}

func validateStartNodeAfterStop(ctx context.Context, t *testing.T, profile string) {
	// TODO (#7496): remove skip once restarts work
	t.Skip("Restarting nodes is broken :(")

	// Grab the stopped node
	name := "m03"

	// Start the node back up
	rr, err := Run(t, exec.CommandContext(ctx, Target(), "-p", profile, "node", "start", name))
	if err != nil {
		t.Errorf("node start returned an error. args %q: %v", rr.Command(), err)
	}

	// Make sure minikube status shows 3 running hosts
	rr, err = Run(t, exec.CommandContext(ctx, Target(), "-p", profile, "status"))
	if err != nil {
		t.Fatalf("failed to run minikube status. args %q : %v", rr.Command(), err)
	}

	if strings.Count(rr.Stdout.String(), "host: Running") != 3 {
		t.Errorf("status says both hosts are not running: args %q: %v", rr.Command(), rr.Stdout.String())
	}

	if strings.Count(rr.Stdout.String(), "kubelet: Running") != 3 {
		t.Errorf("status says both kubelets are not running: args %q: %v", rr.Command(), rr.Stdout.String())
	}
}

func validateDeleteNodeFromMultiNode(ctx context.Context, t *testing.T, profile string) {
	name := "m03"

	// Start the node back up
	rr, err := Run(t, exec.CommandContext(ctx, Target(), "-p", profile, "node", "delete", name))
	if err != nil {
		t.Errorf("node stop returned an error. args %q: %v", rr.Command(), err)
	}

	// Make sure status is back down to 2 hosts
	rr, err = Run(t, exec.CommandContext(ctx, Target(), "-p", profile, "status"))
	if err != nil {
		t.Fatalf("failed to run minikube status. args %q : %v", rr.Command(), err)
	}

	if strings.Count(rr.Stdout.String(), "host: Running") != 2 {
		t.Errorf("status says both hosts are not running: args %q: %v", rr.Command(), rr.Stdout.String())
	}

	if strings.Count(rr.Stdout.String(), "kubelet: Running") != 2 {
		t.Errorf("status says both kubelets are not running: args %q: %v", rr.Command(), rr.Stdout.String())
	}

}