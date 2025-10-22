package oauth

import "context"

// Context keys
type contextKey string

const (
	oauthTokenKey  contextKey = "oauth_token"
	userContextKey contextKey = "user"
)

// WithOAuthToken adds an OAuth token to the context
func WithOAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, oauthTokenKey, token)
}

// GetOAuthToken extracts an OAuth token from the context
func GetOAuthToken(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(oauthTokenKey).(string)
	return token, ok
}

// WithUser adds an authenticated user to context
func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// GetUserFromContext extracts the authenticated user from context.
// Returns the User and true if authentication succeeded, or nil and false otherwise.
//
// Example:
//
//	func toolHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
//	    user, ok := oauth.GetUserFromContext(ctx)
//	    if !ok {
//	        return nil, fmt.Errorf("authentication required")
//	    }
//	    // Use user.Subject, user.Email, user.Username
//	    return mcp.NewToolResultText("Hello, " + user.Username), nil
//	}
func GetUserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userContextKey).(*User)
	return user, ok
}
