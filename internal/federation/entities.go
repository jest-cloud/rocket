package federation

import (
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

			results[i] = entity
		}

		return results, nil
	}
}

