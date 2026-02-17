package federation

import (
	"encoding/json"
	"fmt"

	"github.com/jest-cloud/rocket/internal/types"
)

// EntitiesResolver creates the _entities query resolver for federation
// This resolver is called by the gateway to resolve entity references
func EntitiesResolver(entityResolvers map[string]types.EntityResolveFn) types.FieldResolveFn {
	return func(p types.ResolveParams) (interface{}, error) {
		// Extract representations from arguments
		representations, ok := p.Args["representations"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("_entities query requires 'representations' argument")
		}

		// Resolve each entity representation
		results := make([]interface{}, len(representations))

		for i, rep := range representations {
			repMap, ok := rep.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid representation at index %d: expected map", i)
			}

			// Extract __typename from representation
			typename, ok := repMap["__typename"].(string)
			if !ok {
				return nil, fmt.Errorf("representation at index %d missing __typename field", i)
			}

			// Find entity resolver for this type
			resolver, ok := entityResolvers[typename]
			if !ok {
				return nil, fmt.Errorf("no entity resolver registered for type: %s", typename)
			}

			// Resolve the entity
			entity, err := resolver(p, repMap)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve entity %s: %w", typename, err)
			}

			// Auto-inject __typename for federation compatibility
			entity = injectTypename(entity, typename)

			results[i] = entity
		}

		return results, nil
	}
}

// injectTypename ensures __typename is present in an entity result.
// If the entity is already a map, it sets __typename directly.
// If it's a struct, it converts to a map first via JSON marshaling.
func injectTypename(entity interface{}, typename string) interface{} {
	if entity == nil {
		return entity
	}

	// If already a map, inject directly
	if m, ok := entity.(map[string]interface{}); ok {
		m["__typename"] = typename
		return m
	}

	// Convert struct to map via JSON round-trip
	bytes, err := json.Marshal(entity)
	if err != nil {
		return entity
	}

	var m map[string]interface{}
	if err := json.Unmarshal(bytes, &m); err != nil {
		return entity
	}

	m["__typename"] = typename
	return m
}

