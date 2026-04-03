// Package mcp provides OAuth integration for the official modelcontextprotocol/go-sdk.
//
// This package allows you to add OAuth authentication to MCP servers
// built with github.com/modelcontextprotocol/go-sdk.
//
// Usage:
//
//	import mcpoauth "github.com/tuannvm/oauth-mcp-proxy/mcp"
//
//	mcpServer := mcp.NewServer(&mcp.Implementation{
//	    Name:    "my-server",
//	    Version: "1.0.0",
//	}, nil)
//
//	_, handler, _ := mcpoauth.WithOAuth(mux, &oauth.Config{
//	    Provider: "okta",
//	    Issuer:   "https://company.okta.com",
//	    Audience: "api://your-mcp-server",
//	}, mcpServer)
//
//	http.ListenAndServe(":8080", handler)
//
// Accessing Authenticated User:
//
// In tool handlers, use oauth.GetUserFromContext to access the authenticated user.
// The official SDK also provides auth.TokenInfoFromContext for session management.
//
//	func toolHandler(ctx context.Context, req *mcp.CallToolRequest, params *struct{}) (*mcp.CallToolResult, any, error) {
//	    user, _ := oauth.GetUserFromContext(ctx)
//	    tokenInfo, _ := auth.TokenInfoFromContext(ctx)
//	    // Use user.Subject, user.Email, user.Username
//	    // Use tokenInfo.UserID, tokenInfo.Expiration
//	}
//
// Features:
//
//   - Automatic 401 handling with RFC 6750 compliant WWW-Authenticate headers
//   - auth.TokenInfo population for go-sdk session management
//   - CORS support (OPTIONS pass-through)
//   - Token caching with JWT expiry awareness
//   - All oauth-mcp-proxy security features included
package mcp
