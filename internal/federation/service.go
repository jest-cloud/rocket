package federation

import (
	"github.com/jest-cloud/rocket/internal/types"
)

// ServiceResponse represents the _Service type in federation
type ServiceResponse struct {
	SDL string `json:"sdl"`
}

// ServiceResolver creates the _service query resolver for federation
// This returns the SDL (Schema Definition Language) for the subgraph
func ServiceResolver(schemaSDL string) types.FieldResolveFn {
	return func(p types.ResolveParams) (interface{}, error) {
		return &ServiceResponse{
			SDL: schemaSDL,
		}, nil
	}
}

