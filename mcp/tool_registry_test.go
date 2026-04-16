package mcp_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GoCodeAlone/workflow-plugin-mcp/mcp"
)

func TestToolRegistry_AddAndAll(t *testing.T) {
	reg := mcp.NewToolRegistry("mcp.tool-registry")

	handler := func(_ context.Context, _ *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		return nil, nil
	}

	tool1 := &mcpsdk.Tool{Name: "tool1"}
	tool2 := &mcpsdk.Tool{Name: "tool2"}

	reg.Add(tool1, handler)
	reg.Add(tool2, handler)

	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(all))
	}
	if all[0].Tool.Name != "tool1" {
		t.Errorf("expected first tool name %q, got %q", "tool1", all[0].Tool.Name)
	}
	if all[1].Tool.Name != "tool2" {
		t.Errorf("expected second tool name %q, got %q", "tool2", all[1].Tool.Name)
	}
}

func TestToolRegistry_AllReturnsCopy(t *testing.T) {
	reg := mcp.NewToolRegistry("mcp.tool-registry")

	handler := func(_ context.Context, _ *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		return nil, nil
	}
	tool := &mcpsdk.Tool{Name: "original"}
	reg.Add(tool, handler)

	// Get a copy and mutate it.
	first := reg.All()
	first[0] = mcp.RegisteredTool{Tool: &mcpsdk.Tool{Name: "mutated"}}

	// The registry should be unchanged.
	second := reg.All()
	if len(second) != 1 {
		t.Fatalf("expected 1 tool after mutation, got %d", len(second))
	}
	if second[0].Tool.Name != "original" {
		t.Errorf("registry was mutated; got name %q, want %q", second[0].Tool.Name, "original")
	}
}

func TestToolRegistry_ConcurrentAdd(t *testing.T) {
	const goroutines = 10
	const toolsPerGoroutine = 20

	reg := mcp.NewToolRegistry("mcp.tool-registry")
	handler := func(_ context.Context, _ *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		return nil, nil
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		g := g
		go func() {
			defer wg.Done()
			for i := 0; i < toolsPerGoroutine; i++ {
				reg.Add(&mcpsdk.Tool{Name: fmt.Sprintf("g%d-t%d", g, i)}, handler)
			}
		}()
	}
	wg.Wait()

	all := reg.All()
	want := goroutines * toolsPerGoroutine
	if len(all) != want {
		t.Errorf("expected %d tools after concurrent adds, got %d", want, len(all))
	}
}

func TestToolRegistry_UsesConstructorName(t *testing.T) {
	r := mcp.NewToolRegistry("custom-name")
	if got := r.Name(); got != "custom-name" {
		t.Errorf("Name() = %q, want %q", got, "custom-name")
	}
}
