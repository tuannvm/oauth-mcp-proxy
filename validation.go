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

	// Reject private IPs in redirect URI (prevent SSRF)
	host := parsedURL.Hostname()
	if ip := net.ParseIP(host); ip != nil && !isLocalhostIP(ip) {
		if isPrivateIP(ip) {
			return fmt.Errorf("redirect URI must not point to private IP addresses, got: %s", host)
		}
	}

	return nil
}

// ValidateClientID validates that a client ID is not empty and meets basic requirements.
func ValidateClientID(clientID string) error {
	if clientID == "" {
		return fmt.Errorf("client ID cannot be empty")
	}

	// Check length (OIDC clients typically have reasonable length limits)
	if len(clientID) > 256 {
		return fmt.Errorf("client ID too long (max 256 characters)")
	}

	// Check for whitespace (client IDs shouldn't have whitespace)
	if strings.ContainsAny(clientID, " \t\n\r") {
		return fmt.Errorf("client ID cannot contain whitespace")
	}

	return nil
}

// ValidateClientSecret validates that a client secret meets basic security requirements.
func ValidateClientSecret(clientSecret string) error {
	// Empty is allowed for public clients (no client secret)
	if clientSecret == "" {
		return nil
	}

	// Minimum length for security (OAuth 2.0 recommends at least 128 bits)
	if len(clientSecret) < 16 {
		return fmt.Errorf("client secret too short (minimum 16 characters)")
	}

	// Maximum reasonable length
	if len(clientSecret) > 2048 {
		return fmt.Errorf("client secret too long (max 2048 characters)")
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

// isLocalhostIP checks if an IP is a localhost IP.
func isLocalhostIP(ip net.IP) bool {
	return ip.IsLoopback()
}

// isPrivateIP checks if an IP is in a private range (RFC 1918, RFC 4193, etc.).
func isPrivateIP(ip net.IP) bool {
	// Check for private IPv4 ranges (RFC 1918)
	if ip4 := ip.To4(); ip4 != nil {
		// 10.0.0.0/8
		if ip4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// 169.254.0.0/16 (link-local)
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		return false
	}

	// Check for private IPv6 ranges (RFC 4193, etc.)
	if ip6 := ip.To16(); ip6 != nil {
		// fc00::/7 (unique local)
		if ip6[0]&0xfe == 0xfc {
			return true
		}
		// fe80::/10 (link-local)
		if ip6[0]&0xff == 0xfe && (ip6[1]&0xc0) == 0x80 {
			return true
		}
	}

	return false
}
