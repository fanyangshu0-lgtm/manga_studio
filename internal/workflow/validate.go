package workflow

import (
	"fmt"
	"strings"

	"manga-drama-studio/internal/domain"
)

type Issue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	NodeID  string `json:"nodeId,omitempty"`
	EdgeID  string `json:"edgeId,omitempty"`
}

func Validate(graph domain.Workflow) []Issue {
	issues := []Issue{}
	nodes := map[string]domain.Node{}
	for _, node := range graph.Nodes {
		if strings.TrimSpace(node.ID) == "" {
			issues = append(issues, Issue{Code: "node_id_required", Message: "节点 ID 不能为空"})
			continue
		}
		if _, exists := nodes[node.ID]; exists {
			issues = append(issues, Issue{Code: "duplicate_node", Message: "节点 ID 重复", NodeID: node.ID})
			continue
		}
		if _, ok := definition(node.Type); !ok {
			issues = append(issues, Issue{Code: "unknown_node_type", Message: "未知节点类型: " + node.Type, NodeID: node.ID})
		}
		nodes[node.ID] = node
	}

	incoming := map[string]map[string]bool{}
	adjacency := map[string][]string{}
	edgeKeys := map[string]bool{}
	for _, edge := range graph.Edges {
		source, sourceOK := nodes[edge.Source]
		target, targetOK := nodes[edge.Target]
		if !sourceOK || !targetOK {
			issues = append(issues, Issue{Code: "missing_edge_node", Message: "连线引用了不存在的节点", EdgeID: edge.ID})
			continue
		}
		key := edge.Source + ":" + edge.SourceHandle + ">" + edge.Target + ":" + edge.TargetHandle
		if edgeKeys[key] {
			issues = append(issues, Issue{Code: "duplicate_edge", Message: "存在重复连线", EdgeID: edge.ID})
			continue
		}
		edgeKeys[key] = true
		sourceType, sourcePortOK := outputType(source.Type, edge.SourceHandle)
		targetType, targetPortOK := inputType(target.Type, edge.TargetHandle)
		if !sourcePortOK || !targetPortOK {
			issues = append(issues, Issue{Code: "unknown_port", Message: "连线引用了未知端口", EdgeID: edge.ID})
			continue
		}
		if sourceType != targetType {
			issues = append(issues, Issue{Code: "incompatible_ports", Message: fmt.Sprintf("端口类型不兼容: %s -> %s", sourceType, targetType), EdgeID: edge.ID})
			continue
		}
		if incoming[edge.Target] == nil {
			incoming[edge.Target] = map[string]bool{}
		}
		incoming[edge.Target][edge.TargetHandle] = true
		adjacency[edge.Source] = append(adjacency[edge.Source], edge.Target)
	}

	for _, node := range graph.Nodes {
		def, ok := definition(node.Type)
		if !ok {
			continue
		}
		for _, port := range def.Inputs {
			if port.Required && !incoming[node.ID][port.ID] {
				issues = append(issues, Issue{Code: "required_input", Message: "缺少必填输入: " + port.Label, NodeID: node.ID})
			}
		}
	}
	if hasCycle(nodes, adjacency) {
		issues = append(issues, Issue{Code: "cycle", Message: "工作流不能包含环"})
	}
	return issues
}

func TopologicalOrder(graph domain.Workflow) ([]domain.Node, error) {
	if issues := Validate(graph); len(issues) > 0 {
		return nil, fmt.Errorf("invalid workflow: %s", issues[0].Message)
	}
	byID := map[string]domain.Node{}
	degree := map[string]int{}
	adjacency := map[string][]string{}
	for _, node := range graph.Nodes {
		byID[node.ID] = node
		degree[node.ID] = 0
	}
	for _, edge := range graph.Edges {
		adjacency[edge.Source] = append(adjacency[edge.Source], edge.Target)
		degree[edge.Target]++
	}
	queue := []string{}
	for _, node := range graph.Nodes {
		if degree[node.ID] == 0 {
			queue = append(queue, node.ID)
		}
	}
	order := make([]domain.Node, 0, len(graph.Nodes))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, byID[id])
		for _, next := range adjacency[id] {
			degree[next]--
			if degree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if len(order) != len(graph.Nodes) {
		return nil, fmt.Errorf("workflow contains a cycle")
	}
	return order, nil
}

func inputType(nodeType, portID string) (string, bool) {
	def, ok := definition(nodeType)
	if !ok {
		return "", false
	}
	for _, port := range def.Inputs {
		if port.ID == portID {
			return port.DataType, true
		}
	}
	return "", false
}

func outputType(nodeType, portID string) (string, bool) {
	def, ok := definition(nodeType)
	if !ok {
		return "", false
	}
	for _, port := range def.Outputs {
		if port.ID == portID {
			return port.DataType, true
		}
	}
	return "", false
}

func hasCycle(nodes map[string]domain.Node, adjacency map[string][]string) bool {
	colors := map[string]int{}
	var visit func(string) bool
	visit = func(id string) bool {
		if colors[id] == 1 {
			return true
		}
		if colors[id] == 2 {
			return false
		}
		colors[id] = 1
		for _, next := range adjacency[id] {
			if visit(next) {
				return true
			}
		}
		colors[id] = 2
		return false
	}
	for id := range nodes {
		if visit(id) {
			return true
		}
	}
	return false
}
