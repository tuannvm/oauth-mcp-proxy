package oauth

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateIssuerURL validates that an OIDC issuer URL is properly formatted.
// Enforces HTTPS for non-localhost URLs to prevent MITM attacks.
func ValidateIssuerURL(issuer string) error {
	if issuer == "" {
		return fmt.Errorf("issuer URL cannot be empty")
	}

	// Parse the URL
	parsedURL, err := url.Parse(issuer)
	if err != nil {
		return fmt.Errorf("invalid issuer URL format: %w", err)
	}

	// Must be http or https scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("issuer URL must use http or https scheme, got: %s", parsedURL.Scheme)
	}

	// Must have a host
	if parsedURL.Host == "" {
		return fmt.Errorf("issuer URL must have a host")
	}

	// Enforce HTTPS for non-localhost
	if !isLocalhostHostname(parsedURL.Host) && parsedURL.Scheme != "https" {
		return fmt.Errorf("issuer URL must use HTTPS for non-localhost hosts, got: %s", parsedURL.Scheme)
	}

	// Validate that the hostname is not an IP address (unless localhost)
	// IP addresses in issuer URLs can be problematic for certificate validation
	host := parsedURL.Hostname()
	if net.ParseIP(host) != nil && !isLocalhostHostname(host) {
		return fmt.Errorf("issuer URL hostname should not be a raw IP address (use a domain name instead), got: %s", host)
	}

	// Check for suspicious patterns
	if strings.Contains(host, "..") || strings.HasPrefix(host, ".") {
		return fmt.Errorf("issuer URL hostname contains invalid patterns")
	}

	return nil
}

// ValidateRedirectURI validates a redirect URI for security.
// Checks for proper URL format, scheme, and prevents open redirect vulnerabilities.
func ValidateRedirectURI(redirectURI string) error {
	if redirectURI == "" {
		return fmt.Errorf("redirect URI cannot be empty")
	}

	// Parse the URL
	parsedURL, err := url.Parse(redirectURI)
	if err != nil {
		return fmt.Errorf("invalid redirect URI format: %w", err)
	}

	// Must be http or https scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("redirect URI must use http or https scheme, got: %s", parsedURL.Scheme)
	}

	// Must have a host
	if parsedURL.Host == "" {
		return fmt.Errorf("redirect URI must have a host")
	}

	// Enforce HTTPS for non-localhost
	if !isLocalhostHostname(parsedURL.Host) && parsedURL.Scheme != "https" {
		return fmt.Errorf("redirect URI must use HTTPS for non-localhost hosts, got: %s (use https://)", parsedURL.Scheme)
	}

	// Prevent fragment in redirect URI (OAuth 2.0 security requirement)
	if parsedURL.Fragment != "" {
		return fmt.Errorf("redirect URI must not contain a fragment (per OAuth 2.0 spec)")
	}

	// Check for suspicious patterns in host
	if strings.Contains(parsedURL.Host, "..") || strings.HasPrefix(parsedURL.Host, ".") {
		return fmt.Errorf("redirect URI hostname contains invalid patterns")
	}

	return nil
}

// isLocalhostHostname checks if a hostname is localhost for development.
func isLocalhostHostname(host string) bool {
	// Remove port if present
	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		hostname = host
	}

	// Check for localhost variants
	hostname = strings.ToLower(hostname)
	return hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
}
