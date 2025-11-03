package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/jest-cloud/rocket/internal/registry"
	"github.com/jest-cloud/rocket/internal/types"
	"github.com/wundergraph/astjson"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/plan"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/resolve"
)

// ExecutePlan executes a GraphQL execution plan using graphql-go-tools resolver
func ExecutePlan(ctx context.Context, execPlan plan.Plan, variables map[string]interface{}, resolvers *registry.ResolverRegistry) *types.Result {
	// Convert plan to SynchronousResponsePlan to access Response
	syncPlan, ok := execPlan.(*plan.SynchronousResponsePlan)
	if !ok {
		return &types.Result{
			Errors: []types.Error{{
				Message: "unsupported plan type",
			}},
		}
	}

	// Convert variables to astjson format
	var variablesJSON *astjson.Value
	if len(variables) > 0 {
		variablesBytes, err := json.Marshal(variables)
		if err != nil {
			return &types.Result{
				Errors: []types.Error{{
					Message: fmt.Sprintf("failed to marshal variables: %v", err),
				}},
			}
		}
		variablesJSON, err = astjson.ParseBytes(variablesBytes)
		if err != nil {
			return &types.Result{
				Errors: []types.Error{{
					Message: fmt.Sprintf("failed to parse variables: %v", err),
				}},
			}
		}
	}

	// Create resolver context
	resolverCtx := resolve.NewContext(ctx)
	resolverCtx.Variables = variablesJSON
	
	// Create resolver instance
	resolver := resolve.New(ctx, resolve.ResolverOptions{
		ResolvableOptions: resolve.ResolvableOptions{
			ApolloCompatibilityValueCompletionInExtensions: false,
			ApolloCompatibilityTruncateFloatValues:         false,
			ApolloCompatibilitySuppressFetchErrors:         false,
			ApolloCompatibilityReplaceInvalidVarError:      false,
		},
	})

	// Build Fetches tree from RawFetches if Fetches is nil
	// Field coordinates are already set during planning in ConfigureFetch()
	if syncPlan.Response.Fetches == nil && len(syncPlan.Response.RawFetches) > 0 {
		if len(syncPlan.Response.RawFetches) == 1 {
			fetchItem := syncPlan.Response.RawFetches[0]
			if fetchItem.Fetch != nil {
				syncPlan.Response.Fetches = resolve.Single(fetchItem.Fetch)
			}
		} else {
			fetches := make([]*resolve.FetchTreeNode, 0, len(syncPlan.Response.RawFetches))
			for _, fetchItem := range syncPlan.Response.RawFetches {
				if fetchItem.Fetch != nil {
					fetches = append(fetches, resolve.Single(fetchItem.Fetch))
				}
			}
			if len(fetches) > 0 {
				syncPlan.Response.Fetches = resolve.Parallel(fetches...)
			}
		}
	}
	
	var output bytes.Buffer
	_, err := resolver.ResolveGraphQLResponse(resolverCtx, syncPlan.Response, nil, &output)
	
	if err != nil {
		return &types.Result{
			Errors: []types.Error{{
				Message: fmt.Sprintf("failed to resolve query: %v", err),
			}},
		}
	}
	
	// Handle empty output
	if output.Len() == 0 {
		return &types.Result{
			Data:   map[string]interface{}{},
			Errors: []types.Error{},
		}
	}

	// Parse the output JSON
	var result map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		return &types.Result{
			Errors: []types.Error{{
				Message: fmt.Sprintf("failed to parse result: %v, raw output: %s", err, output.String()),
			}},
		}
	}

	// Extract data and errors from result
	var data map[string]interface{}
	if resultData, ok := result["data"]; ok && resultData != nil {
		if dataMap, ok := resultData.(map[string]interface{}); ok {
			data = dataMap
		}
		// If data is not a map, leave it as nil - it might be a scalar/null
	}
	
	var errors []types.Error
	if errs, ok := result["errors"].([]interface{}); ok {
		errors = make([]types.Error, len(errs))
		for i, e := range errs {
			if errMap, ok := e.(map[string]interface{}); ok {
				message, _ := errMap["message"].(string)
				errors[i] = types.Error{
					Message: message,
				}
				if path, ok := errMap["path"].([]interface{}); ok {
					errors[i].Path = path
				}
			}
		}
	}

	return &types.Result{
		Data:   data,
		Errors: errors,
	}
}

