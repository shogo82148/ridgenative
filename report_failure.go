//go:build !go1.27

package ridgenative

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// reportFailure reports the error to the Runtime API.
func (c *runtimeAPIClient) reportFailure(ctx context.Context, invoke *invoke, invokeErr *invokeResponseError) error {
	body, err := json.Marshal(invokeErr)
	if err != nil {
		return fmt.Errorf("ridgenative: failed to marshal the function error: %w", err)
	}
	log.Printf("%s", body)
	if err := c.post(ctx, invoke.id+"/error", body, contentTypeJSON); err != nil {
		return fmt.Errorf("ridgenative: unexpected error occurred when sending the function error to the API: %w", err)
	}
	return nil
}
