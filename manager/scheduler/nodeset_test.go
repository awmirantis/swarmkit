package scheduler

import (
	"testing"

	"github.com/moby/swarmkit/v2/api"
)

func TestTreeTaskCountConsistency(t *testing.T) {
	// Create a nodeSet with some test nodes
	ns := &nodeSet{nodes: make(map[string]NodeInfo)}

	// Add test nodes with different labels and task counts
	nodes := []NodeInfo{
		{
			Node: &api.Node{
				ID: "node1",
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Labels: map[string]string{"datacenter": "dc1", "rack": "r1"},
					},
				},
			},
			ActiveTasksCountByService: map[string]int{"service1": 3},
		},
		{
			Node: &api.Node{
				ID: "node2",
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Labels: map[string]string{"datacenter": "dc1", "rack": "r2"},
					},
				},
			},
			ActiveTasksCountByService: map[string]int{"service1": 2},
		},
		{
			Node: &api.Node{
				ID: "node3",
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Labels: map[string]string{"datacenter": "dc2", "rack": "r2"},
					},
				},
			},
			ActiveTasksCountByService: map[string]int{"service1": 4},
		},
		{
			Node: &api.Node{
				ID: "node4",
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Labels: map[string]string{}, // no label
					},
				},
			},
			ActiveTasksCountByService: map[string]int{"service1": 2},
		},
		{
			Node: &api.Node{
				ID: "node5",
				Spec: api.NodeSpec{
					Annotations: api.Annotations{
						Labels: map[string]string{}, // no label
					},
				},
			},
			ActiveTasksCountByService: map[string]int{"service1": 1},
		},
	}

	for _, node := range nodes {
		ns.addOrUpdateNode(node)
	}

	preferences := []*api.PlacementPreference{
		{
			Preference: &api.PlacementPreference_Spread{
				Spread: &api.SpreadOver{
					SpreadDescriptor: "node.labels.datacenter",
				},
			},
		},
		{
			Preference: &api.PlacementPreference_Spread{
				Spread: &api.SpreadOver{
					SpreadDescriptor: "node.labels.rack",
				},
			},
		},
	}

	// Create the tree
	tree := ns.tree("service1", preferences, 10,
		func(*NodeInfo) bool { return true },
		func(a, b *NodeInfo) bool { return true })

	// Helper function to verify task count consistency recursively
	var verifyTaskCounts func(*testing.T, *decisionTree) int
	verifyTaskCounts = func(t *testing.T, dt *decisionTree) int {
		if dt == nil {
			return 0
		}

		if dt.next == nil {
			return dt.tasks
		}

		// Calculate sum of children's tasks
		childrenSum := 0
		for _, child := range dt.next {
			childrenSum += verifyTaskCounts(t, child)
		}

		// Verify parent's task count equals sum of children
		if dt.tasks != childrenSum {
			t.Errorf("Parent task count (%d) does not equal sum of children (%d)",
				dt.tasks, childrenSum)
		}

		return dt.tasks
	}

	// Run the verification
	verifyTaskCounts(t, &tree)

	// Verify specific expected values
	if tree.tasks != 12 { // Total tasks: 3 + 2 + 4 + 2 + 1 = 12
		t.Errorf("Expected root to have 12 tasks, got %d", tree.tasks)
	}

	dc1Tasks := tree.next["dc1"].tasks
	if dc1Tasks != 5 { // dc1 tasks: 3 + 2 = 5
		t.Errorf("Expected dc1 to have 5 tasks, got %d", dc1Tasks)
	}
	dc1r1Tasks := tree.next["dc1"].next["r1"].tasks
	if dc1r1Tasks != 3 {
		t.Errorf("Expected dc1 r1 to have 3 tasks, got %d", dc1r1Tasks)
	}
	dc1r2Tasks := tree.next["dc1"].next["r2"].tasks
	if dc1r2Tasks != 2 {
		t.Errorf("Expected dc1 r1 to have 2 tasks, got %d", dc1r2Tasks)
	}

	dc2Tasks := tree.next["dc2"].tasks
	if dc2Tasks != 4 { // dc2 tasks: 4
		t.Errorf("Expected dc2 to have 4 tasks, got %d", dc2Tasks)
	}
	dc2r2Tasks := tree.next["dc2"].next["r2"].tasks
	if dc2r2Tasks != 4 {
		t.Errorf("Expected dc1 r1 to have 4 tasks, got %d", dc1r2Tasks)
	}

	otherTasks := tree.next[""].tasks
	if otherTasks != 3 {
		t.Errorf("Expected others to have 3 tasks, got %d", otherTasks)
	}
	subOtherTasks := tree.next[""].next[""].tasks
	if subOtherTasks != 3 {
		t.Errorf("Expected sub-others to have 3 tasks, got %d", subOtherTasks)
	}

}
