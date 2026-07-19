package workflow

import (
	"testing"

	"manga-drama-studio/internal/domain"
)

func TestExampleIsValid(t *testing.T) {
	if issues := Validate(Example("project")); len(issues) != 0 {
		t.Fatalf("example should be valid: %#v", issues)
	}
}

func TestValidateRejectsCycle(t *testing.T) {
	graph := Example("project")
	graph.Edges = append(graph.Edges, domain.Edge{ID: "cycle", Source: "compose", SourceHandle: "video", Target: "script", TargetHandle: "story"})
	issues := Validate(graph)
	found := false
	for _, issue := range issues {
		if issue.Code == "incompatible_ports" || issue.Code == "cycle" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected invalid edge or cycle issue: %#v", issues)
	}
}

func TestValidateRejectsMissingRequiredInput(t *testing.T) {
	graph := Example("project")
	graph.Edges = graph.Edges[1:]
	issues := Validate(graph)
	found := false
	for _, issue := range issues {
		if issue.Code == "required_input" && issue.NodeID == "script" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected required input issue: %#v", issues)
	}
}
