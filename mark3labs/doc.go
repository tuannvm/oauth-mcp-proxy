// Package mark3labs provides OAuth integration for the mark3labs/mcp-go SDK.
//
// This package allows you to add OAuth authentication to MCP servers
// built with github.com/mark3labs/mcp-go.
//
// Usage:
//
//	import "github.com/tuannvm/oauth-mcp-proxy/mark3labs"
//
//	oauthServer, oauthOption, _ := mark3labs.WithOAuth(mux, &oauth.Config{
//	    Provider: "okta",
//	    Issuer:   "https://company.okta.com",
//	    Audience: "api://your-mcp-server",
//	})
//	mcpServer := server.NewMCPServer("Server", "1.0.0", oauthOption)
//	streamableServer := server.NewStreamableHTTPServer(mcpServer, ...)
//	mux.HandleFunc("/mcp", oauthServer.WrapMCPEndpoint(streamableServer))
//
// Accessing Authenticated User:
//
//	func toolHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
//	    user, ok := oauth.GetUserFromContext(ctx)
//	    if !ok {
//	        return nil, fmt.Errorf("authentication required")
//	    }
//	    // Use user.Subject, user.Email, user.Username
//	}
package mark3labs
