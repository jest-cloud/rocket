package rocket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/wundergraph/astjson"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/plan"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/resolve"
)

// ExecutePlan executes a GraphQL execution plan using graphql-go-tools resolver
func ExecutePlan(ctx context.Context, execPlan plan.Plan, variables map[string]interface{}, resolvers *ResolverRegistry) *Result {
	// Convert plan to SynchronousResponsePlan to access Response
	syncPlan, ok := execPlan.(*plan.SynchronousResponsePlan)
	if !ok {
		return &Result{
			Errors: []Error{{
				Message: "unsupported plan type",
			}},
		}
	}

	// Convert variables to astjson format
	var variablesJSON *astjson.Value
	if len(variables) > 0 {
		variablesBytes, err := json.Marshal(variables)
		if err != nil {
			return &Result{
				Errors: []Error{{
					Message: fmt.Sprintf("failed to marshal variables: %v", err),
				}},
			}
		}
		variablesJSON, err = astjson.ParseBytes(variablesBytes)
		if err != nil {
			return &Result{
				Errors: []Error{{
					Message: fmt.Sprintf("failed to parse variables: %v", err),
				}},
			}
		}
	}

	// Create resolver context using NewContext
	resolverCtx := resolve.NewContext(ctx)
	resolverCtx.Variables = variablesJSON

	// Create resolver instance - New takes context and ResolverOptions
	resolver := resolve.New(ctx, resolve.ResolverOptions{
		ResolvableOptions: resolve.ResolvableOptions{
			ApolloCompatibilityValueCompletionInExtensions: false,
			ApolloCompatibilityTruncateFloatValues:         false,
			ApolloCompatibilitySuppressFetchErrors:         false,
			ApolloCompatibilityReplaceInvalidVarError:      false,
		},
	})
	
	// Build Fetches tree from RawFetches if Fetches is nil
	// The planner populates RawFetches, but ResolveGraphQLResponse needs Fetches tree
	// This is normally done by a PostProcessor, but we need to do it manually
	if syncPlan.Response.Fetches == nil && len(syncPlan.Response.RawFetches) > 0 {
		// For a single fetch, use Single() helper
		if len(syncPlan.Response.RawFetches) == 1 {
			fetchItem := syncPlan.Response.RawFetches[0]
			if fetchItem.Fetch != nil {
				syncPlan.Response.Fetches = resolve.Single(fetchItem.Fetch)
			}
		} else {
			// For multiple fetches, use Parallel() helper
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

	// Extract field coordinates from FetchInfo and store them in RocketSource
	// This must happen BEFORE ResolveGraphQLResponse calls Load
	for _, fetch := range syncPlan.Response.RawFetches {
		if fetch.Fetch != nil {
			if singleFetch, ok := fetch.Fetch.(*resolve.SingleFetch); ok {
				if info := singleFetch.FetchInfo(); info != nil {
					// Extract field coordinate and store it in RocketSource
					if rocketSource, ok := singleFetch.DataSource.(*RocketSource); ok && len(info.RootFields) > 0 {
						coord := info.RootFields[0]
						rocketSource.fieldCoord = &fieldCoordinate{
							TypeName:  coord.TypeName,
							FieldName: coord.FieldName,
						}
					}
				}
			}
		}
	}
	
	var output bytes.Buffer
	_, err := resolver.ResolveGraphQLResponse(resolverCtx, syncPlan.Response, nil, &output)
	if err != nil {
		return &Result{
			Errors: []Error{{
				Message: fmt.Sprintf("failed to resolve query: %v", err),
			}},
		}
	}

	// Parse the output JSON
	var result map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		return &Result{
			Errors: []Error{{
				Message: fmt.Sprintf("failed to parse result: %v", err),
			}},
		}
	}

	// Extract data and errors from result
	data, _ := result["data"].(map[string]interface{})
	var errors []Error
	if errs, ok := result["errors"].([]interface{}); ok {
		errors = make([]Error, len(errs))
		for i, e := range errs {
			if errMap, ok := e.(map[string]interface{}); ok {
				message, _ := errMap["message"].(string)
				errors[i] = Error{
					Message: message,
				}
				if path, ok := errMap["path"].([]interface{}); ok {
					errors[i].Path = path
				}
			}
		}
	}

	return &Result{
		Data:   data,
		Errors: errors,
	}
}

