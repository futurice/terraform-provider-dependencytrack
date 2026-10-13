// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package configproperty

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// newV5Resource returns the resource with a client pinned to 5.2.0 whose server fails the test on any request.
func newV5Resource(t *testing.T) (*ConfigPropertyResource, tfsdk.State) {
	t.Helper()
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client, err := dtrack.NewClient(server.URL, dtrack.WithServerVersion("5.2.0"))
	if err != nil {
		t.Fatal(err)
	}
	r := &ConfigPropertyResource{client: client}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	// State as a v4 apply leaves it, which the migration carries over to v5.
	state := tfsdk.State{Schema: schemaResp.Schema}
	diags := state.Set(ctx, &ConfigPropertyResourceModel{
		ID:            types.StringValue("scanner/trivy.enabled"),
		GroupName:     types.StringValue("scanner"),
		Name:          types.StringValue("trivy.enabled"),
		Value:         types.StringValue("true"),
		DestroyValue:  types.StringNull(),
		OriginalValue: types.StringValue("false"),
	})
	if diags.HasError() {
		t.Fatalf("failed to build state: %v", diags)
	}

	return r, state
}

func TestReadOnV5RemovesStateWithoutRequests(t *testing.T) {
	r, state := newV5Resource(t)

	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read failed: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("Read kept the resource in state on v5")
	}
}

func TestDeleteOnV5RemovesStateWithoutRequests(t *testing.T) {
	r, state := newV5Resource(t)

	resp := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete failed: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("Delete kept the resource in state on v5")
	}
}
