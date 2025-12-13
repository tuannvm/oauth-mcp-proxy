package mark3labs

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	oauth "github.com/tuannvm/oauth-mcp-proxy"
)

// NewMiddleware creates an authentication middleware for mark3labs/mcp-go SDK.
// It validates OAuth tokens, caches results, and adds authenticated user to context.
//
// The middleware:
//  1. Extracts OAuth token from context (set by CreateHTTPContextFunc)
//  2. Validates token using Server.ValidateTokenCached (with 5-minute cache)
//  3. Adds User to context via oauth.WithUser
//  4. Passes request to tool handler with authenticated context
//
// Use oauth.GetUserFromContext(ctx) in tool handlers to access authenticated user.
func NewMiddleware(s *oauth.Server) func(server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			tokenString, ok := oauth.GetOAuthToken(ctx)
			if !ok {
				return nil, fmt.Errorf("authentication required: missing OAuth token")
			}

			user, err := s.ValidateTokenCached(ctx, tokenString)
			if err != nil {
				return nil, err
			}

			ctx = oauth.WithUser(ctx, user)

			return next(ctx, req)
		}
	}
}
