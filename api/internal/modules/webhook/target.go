package webhook

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/zgiai/luas/api/internal/capabilities/crypto"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
)

const maxWebhookTargetBytes = 2048

type ipResolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type networkDialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

type targetPolicy struct {
	resolver           ipResolver
	dialer             networkDialer
	allowInsecureHTTP  bool
	allowPrivateTarget bool
}

// NewTargetPolicy creates one URL and network-policy authority for setup and delivery.
func NewTargetPolicy(cfg *config.Config) *targetPolicy {
	policy := &targetPolicy{
		resolver: net.DefaultResolver,
		dialer:   &net.Dialer{},
	}
	if cfg != nil {
		policy.allowInsecureHTTP = cfg.Webhook.AllowInsecureHTTP
		policy.allowPrivateTarget = cfg.Webhook.AllowPrivateTargets
	}
	return policy
}

func (p *targetPolicy) Normalize(ctx context.Context, value string) (string, string, error) {
	normalized, parsed, err := p.normalizeSyntax(value)
	if err != nil {
		return "", "", err
	}
	if _, err := p.resolveAllowed(ctx, parsed.Hostname()); err != nil {
		return "", "", err
	}
	return normalized, crypto.SHA256Hex(normalized), nil
}

func (p *targetPolicy) normalizeSyntax(value string) (string, *url.URL, error) {
	if p == nil {
		return "", nil, domain.ErrServiceUnavailable
	}
	if value == "" || value != strings.TrimSpace(value) || len(value) > maxWebhookTargetBytes || containsUnsafeURLRune(value) {
		return "", nil, domain.ErrWebhookInvalidTarget
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || !parsed.IsAbs() || parsed.Opaque != "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return "", nil, domain.ErrWebhookInvalidTarget
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "https" && (parsed.Scheme != "http" || !p.allowInsecureHTTP) {
		return "", nil, domain.ErrWebhookInvalidTarget
	}

	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" || strings.HasSuffix(hostname, ".") || strings.Contains(hostname, "%") || !asciiHostname(hostname) {
		return "", nil, domain.ErrWebhookInvalidTarget
	}
	port := parsed.Port()
	if port != "" {
		value, parseErr := strconv.ParseUint(port, 10, 16)
		if parseErr != nil || value == 0 {
			return "", nil, domain.ErrWebhookInvalidTarget
		}
	}
	if (parsed.Scheme == "https" && port == "443") || (parsed.Scheme == "http" && port == "80") {
		port = ""
	}
	if strings.Contains(hostname, ":") {
		parsed.Host = "[" + hostname + "]"
	} else {
		parsed.Host = hostname
	}
	if port != "" {
		parsed.Host = net.JoinHostPort(hostname, port)
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	parsed.RawPath = ""
	normalized := parsed.String()
	if len(normalized) > maxWebhookTargetBytes {
		return "", nil, domain.ErrWebhookInvalidTarget
	}
	return normalized, parsed, nil
}

func (p *targetPolicy) resolveAllowed(ctx context.Context, hostname string) ([]netip.Addr, error) {
	if p == nil || p.resolver == nil {
		return nil, domain.ErrServiceUnavailable
	}
	if parsed, err := netip.ParseAddr(hostname); err == nil {
		parsed = parsed.Unmap()
		if !p.allowedAddress(parsed) {
			return nil, domain.ErrWebhookInvalidTarget
		}
		return []netip.Addr{parsed}, nil
	}
	addresses, err := p.resolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return nil, fmt.Errorf("resolve webhook target: %w", err)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("resolve webhook target: %w", errNoWebhookTargetAddress)
	}
	result := make([]netip.Addr, 0, len(addresses))
	seen := make(map[netip.Addr]struct{}, len(addresses))
	for _, address := range addresses {
		parsed, ok := netip.AddrFromSlice(address.IP)
		if !ok {
			return nil, domain.ErrWebhookInvalidTarget
		}
		parsed = parsed.Unmap()
		if !p.allowedAddress(parsed) {
			return nil, domain.ErrWebhookInvalidTarget
		}
		if _, duplicate := seen[parsed]; duplicate {
			continue
		}
		seen[parsed] = struct{}{}
		result = append(result, parsed)
	}
	if len(result) == 0 {
		return nil, domain.ErrWebhookInvalidTarget
	}
	return result, nil
}

func (p *targetPolicy) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if p == nil || p.dialer == nil {
		return nil, domain.ErrServiceUnavailable
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil || port == "" {
		return nil, domain.ErrWebhookInvalidTarget
	}
	addresses, err := p.resolveAllowed(ctx, host)
	if err != nil {
		return nil, err
	}
	var dialErrors []error
	for _, candidate := range addresses {
		connection, dialErr := p.dialer.DialContext(ctx, network, net.JoinHostPort(candidate.String(), port))
		if dialErr == nil {
			return connection, nil
		}
		dialErrors = append(dialErrors, dialErr)
	}
	return nil, errors.Join(dialErrors...)
}

func (p *targetPolicy) allowedAddress(address netip.Addr) bool {
	if !address.IsValid() || address.IsUnspecified() || address.IsMulticast() {
		return false
	}
	if p.allowPrivateTarget {
		return true
	}
	if !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() {
		return false
	}
	for _, prefix := range blockedWebhookPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func asciiHostname(value string) bool {
	for _, char := range value {
		if char > unicode.MaxASCII || char <= 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func containsUnsafeURLRune(value string) bool {
	for _, char := range value {
		if char == '\\' || char <= 0x20 || char == 0x7f {
			return true
		}
	}
	return false
}

func mustWebhookPrefix(value string) netip.Prefix {
	return netip.MustParsePrefix(value)
}

var (
	errNoWebhookTargetAddress = errors.New("webhook target has no address")
	blockedWebhookPrefixes    = []netip.Prefix{
		mustWebhookPrefix("0.0.0.0/8"),
		mustWebhookPrefix("100.64.0.0/10"),
		mustWebhookPrefix("192.0.0.0/24"),
		mustWebhookPrefix("192.0.2.0/24"),
		mustWebhookPrefix("198.18.0.0/15"),
		mustWebhookPrefix("198.51.100.0/24"),
		mustWebhookPrefix("203.0.113.0/24"),
		mustWebhookPrefix("240.0.0.0/4"),
		mustWebhookPrefix("100::/64"),
		mustWebhookPrefix("2001:db8::/32"),
	}
)
