#!/usr/bin/env python3

"""Keep outbound webhooks finite, signed, SSRF-safe, durable, and private."""

from __future__ import annotations

import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]


def read(relative_path: str) -> str:
    path = ROOT / relative_path
    if not path.exists():
        raise FileNotFoundError(relative_path)
    return path.read_text(encoding="utf-8")


def require_all(
    failures: list[str], relative_path: str, markers: tuple[str, ...]
) -> None:
    try:
        content = read(relative_path)
    except FileNotFoundError:
        failures.append(f"{relative_path} is missing")
        return
    for marker in markers:
        if marker not in content:
            failures.append(f"{relative_path} must contain {marker!r}")


def forbid_any(
    failures: list[str], relative_path: str, markers: tuple[str, ...]
) -> None:
    try:
        content = read(relative_path)
    except FileNotFoundError:
        failures.append(f"{relative_path} is missing")
        return
    for marker in markers:
        if marker in content:
            failures.append(f"{relative_path} must not contain {marker!r}")


def check_web_routes(failures: list[str]) -> None:
    route_root = ROOT / "web" / "src" / "app" / "api"
    actual = {
        path.relative_to(route_root).as_posix()
        for path in route_root.glob("webhook-*/**/route.ts")
    }
    expected = {
        "webhook-event-types/route.ts",
        "webhook-endpoints/route.ts",
        "webhook-endpoints/[id]/route.ts",
        "webhook-endpoints/[id]/status/route.ts",
        "webhook-endpoints/[id]/secret-rotations/route.ts",
        "webhook-endpoints/[id]/tests/route.ts",
        "webhook-deliveries/route.ts",
        "webhook-deliveries/[id]/attempts/route.ts",
    }
    if actual != expected:
        missing = sorted(expected - actual)
        unexpected = sorted(actual - expected)
        if missing:
            failures.append(f"Web webhook routes are missing: {', '.join(missing)}")
        if unexpected:
            failures.append(
                "Web webhook routes must remain an explicit allowlist; unexpected: "
                + ", ".join(unexpected)
            )


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "contracts/WEBHOOKS.md",
        (
            "optional `webhook` starter",
            "OPTIONAL_STARTERS=organization,webhook",
            "NEXT_PUBLIC_OPTIONAL_FEATURES=organization,webhook",
            "Standard Webhooks",
            "shipped catalog contains only `webhook.test`",
            "The browser cannot publish arbitrary events",
            "redirects are never followed",
            "DNS is resolved again at delivery time",
            "same lease token",
            "most ten attempts",
            "never store response bodies",
            "Deliberate Deferrals",
        ),
    )
    require_all(
        failures,
        "CONTEXT.md",
        (
            "**Webhook endpoint**",
            "**Webhook event**",
            "**Webhook delivery**",
            "**webhook vs event bus/workflow/inbound callback**",
        ),
    )
    require_all(
        failures,
        "api/internal/domain/webhook.go",
        (
            "type WebhookPublisher interface",
            "type WebhookDispatcher interface",
            "type WebhookTester interface",
            "type WebhookMaintainer interface",
            "PublishWebhook(context.Context, WebhookEvent)",
            "DispatchWebhooks(context.Context, int)",
        ),
    )
    require_all(
        failures,
        "api/internal/domain/error_codes.go",
        (
            '"WEBHOOK.ENDPOINT_NOT_FOUND"',
            '"WEBHOOK.DELIVERY_NOT_FOUND"',
            '"WEBHOOK.IDEMPOTENCY_CONFLICT"',
            '"WEBHOOK.ENDPOINT_VERSION_CONFLICT"',
            '"WEBHOOK.REPLAY_NOT_ALLOWED"',
            '"WEBHOOK.INVALID_EVENT_TYPE"',
            '"WEBHOOK.INVALID_TARGET"',
            '"WEBHOOK.PRECONDITION_REQUIRED"',
        ),
    )
    require_all(
        failures,
        "api/internal/bootstrap/domain_error_mappings.go",
        (
            "ErrWebhookEndpointNotFound, http.StatusNotFound",
            "ErrWebhookDeliveryNotFound, http.StatusNotFound",
            "ErrWebhookIdempotencyConflict, http.StatusConflict",
            "ErrWebhookEndpointVersionConflict, http.StatusConflict",
            "ErrWebhookReplayNotAllowed, http.StatusConflict",
            "ErrWebhookInvalidEventType, http.StatusUnprocessableEntity",
            "ErrWebhookInvalidTarget, http.StatusUnprocessableEntity",
            "ErrWebhookPreconditionRequired, http.StatusPreconditionRequired",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/provider.go",
        (
            '"webhook"',
            'WithStarterDependencies("user", "audit", "organization")',
            "2026_07_15_060000_create_webhook_tables",
            "wire.Bind(new(domain.WebhookPublisher)",
            "wire.Bind(new(domain.WebhookDispatcher)",
            "wire.Bind(new(domain.WebhookTester)",
            "wire.Bind(new(domain.WebhookMaintainer)",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/config/config.go",
        (
            'env.Get("WEBHOOK_ENCRYPTION_KEY", "")',
            'env.GetDuration("WEBHOOK_REQUEST_TIMEOUT", DefaultWebhookRequestTimeout)',
            "WEBHOOK_REQUEST_TIMEOUT must be greater than 0 and no more than 30s",
            "WEBHOOK_ALLOW_INSECURE_HTTP cannot be enabled in production",
            "WEBHOOK_ALLOW_PRIVATE_TARGETS cannot be enabled in production",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/catalog.go",
        (
            "maxWebhookDefinitions    = 128",
            "maxWebhookPayloadBytes   = 64 * 1024",
            'Type:            "webhook.test"',
            "validateTestPayload",
            "json.Decoder",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/target.go",
        (
            "parsed.User != nil",
            "parsed.RawQuery != \"\"",
            "parsed.Fragment != \"\"",
            "LookupIPAddr(ctx, hostname)",
            "for _, address := range addresses",
            "if !p.allowedAddress(parsed)",
            "net.JoinHostPort(candidate.String(), port)",
            "address.IsPrivate()",
            "address.IsLoopback()",
            "address.IsLinkLocalUnicast()",
            "blockedWebhookPrefixes",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/sender.go",
        (
            'request.Header.Set("webhook-id"',
            'request.Header.Set("webhook-timestamp"',
            'request.Header.Set("webhook-signature"',
            'request.Header.Set("User-Agent", webhookUserAgent)',
            "Proxy:                 nil",
            "DialContext:           s.policy.DialContext",
            "CheckRedirect:",
            "errWebhookRedirectBlocked",
            "io.Copy(io.Discard, io.LimitReader",
            "webhookHTTPFailureCode(status)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/secret.go",
        (
            'webhookSecretPrefix = "whsec_"',
            "crypto.GenerateKey(32)",
            "crypto.NewAESEncryptorFromString",
            "EncryptString(plaintext)",
            "DecryptString(ciphertext)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/model.go",
        (
            "SecretCiphertext",
            "PreviousSecretCiphertext",
            "DestinationHash",
            "LeaseToken",
            "FailureCode",
            "ResponseTruncated",
            "webhook_delivery_attempts",
            "idx_webhook_deliveries_due",
            "idx_webhook_deliveries_lease_expiry",
            "idx_webhook_deliveries_organization_endpoint_created",
        ),
    )
    forbid_any(
        failures,
        "api/internal/modules/webhook/model.go",
        ("ResponseBody", "ErrorMessage", "RequestSignature", "DNSAnswer"),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/repository.go",
        (
            "ResolveContextDB",
            "Fingerprint != mutation.Fingerprint",
            "clause.OnConflict{",
            "DoNothing: true",
            'clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}',
            'Where("id = ? AND status = ? AND lease_token = ?"',
            "cancelOpenDeliveries",
            "ConsecutiveFailures + 1",
            "func (r *repository) Prune(",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/service.go",
        (
            "webhookMaximumAttempts",
            "webhookDisableAfterFailure",
            "func (s *service) PublishWebhookTest(",
            'Source:         "webhook.http_test"',
            'Type:           "webhook.test"',
            "Occurrence time is server-generated",
            "webhookRetryDelay(",
            "s.store.Complete(ctx, claim, completion, webhookDisableAfterFailure)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/handler.go",
        (
            "maxWebhookManagementBodyBytes = 16 * 1024",
            "CanManageOrganization()",
            'c.Request.Header.Values("If-Match")',
            'c.Request.Header.Values("Idempotency-Key")',
            "http.MaxBytesReader",
            'c.Header("Cache-Control", "private, no-store")',
            'c.Header("Vary", "Authorization, Organization-Id")',
        ),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/routes.go",
        (
            'auth.GET("/webhook-event-types"',
            'auth.POST("/webhook-endpoints"',
            'auth.POST("/webhook-endpoints/:id/tests"',
            'auth.GET("/webhook-deliveries"',
            'auth.GET("/webhook-deliveries/:id/attempts"',
            'auth.WithMiddleware("auth", "organization_context")',
        ),
    )
    require_all(
        failures,
        "api/database/migrations/2026_07_15_060000_create_webhook_tables.go",
        (
            "UseTransaction: true",
            "webhook.EndpointPO{}",
            "webhook.SubscriptionPO{}",
            "webhook.EventPO{}",
            "webhook.DeliveryPO{}",
            "webhook.AttemptPO{}",
            "DropTable(",
        ),
    )
    require_all(
        failures,
        "api/internal/bootstrap/operatorcommands/webhook.go",
        (
            'return "webhook:work"',
            'return "webhook:publish-test"',
            'return "webhook:replay"',
            'return "webhook:prune"',
            "signal.NotifyContext",
            "application.AuditRecorder.Record(",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/service_test.go",
        (
            "TestServiceCreatesPublishesAndDispatchesWebhook",
            "TestServiceRetriesWithStableMessageID",
            "TestServiceRotatesAndVersionProtectsEndpoint",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/repository_test.go",
        (
            "TestRepositoryDurableDeliveryLifecycle",
            "TestRepositoryAutoDisablesEndpointAfterConsecutiveTerminalFailures",
            "TestRepositoryVersionedMutationCancelsOpenWorkAndScrubsSecrets",
            "TestRepositoryPublicationHonorsOuterTransaction",
            "TestWebhookLockQueriesSkipOnlyContendedClaimRows",
            "TestWebhookIndexesMatchWorkerAndManagementQueries",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/target_test.go",
        (
            "TestTargetPolicyRejectsUnsafeURLFormsAndNetworks",
            "TestTargetPolicyNormalizesAndRequiresEveryResolvedAddressToBePublic",
            "TestTargetPolicyDialsOnlyTheVerifiedAddress",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/webhook/sender_test.go",
        (
            "TestSenderSignsTheExactBodyAndHeaders",
            "TestSenderDoesNotFollowRedirects",
            "TestSenderBoundsResponsesAndClassifiesRetryableFailures",
            "TestSenderClassifiesIncompleteResponseBodyAsRetryableNetworkFailure",
            "TestSenderDurationIncludesResponseBodyDrain",
            "TestSenderClassifiesTimeoutWithoutPersistingProviderText",
        ),
    )

    require_all(
        failures,
        "web/src/config/optional-features.ts",
        ("'webhook'", "webhook: ['organization']"),
    )
    require_all(
        failures,
        "web/src/features/webhook/schemas.ts",
        (
            "from 'zod/mini'",
            "literal('webhook.test')",
            "strictObject",
            "webhookEndpointSecretSchema",
            "webhookDeliverySchema",
            "webhookAttemptSchema",
        ),
    )
    require_all(
        failures,
        "web/src/features/webhook/server/webhook-route.ts",
        (
            "guardSameOriginMutation(request)",
            "authenticateOrganizationBackend",
            "context.role !== 'owner' && context.role !== 'admin'",
            "readJsonBody(request, maxManagementBodyBytes)",
            "path: 'webhook-event-types'",
            "path: 'webhook-endpoints'",
            "path: `webhook-endpoints/${endpointId}/tests`",
            "path: 'webhook-deliveries'",
            "path: `webhook-deliveries/${deliveryId}/attempts`",
            "privateNoStoreResponse(response, ['Cookie', 'Organization-Id'])",
            "idempotencyKey: idempotencyKey.value",
        ),
    )
    require_all(
        failures,
        "web/src/features/webhook/server/mock-webhook-store.ts",
        (
            "eventTypes(): readonly ['webhook.test']",
            "signing_secret: signingSecret",
            "WEBHOOK.MOCK_NOT_DELIVERED",
            "status: 'canceled'",
        ),
    )
    forbid_any(
        failures,
        "web/src/features/webhook/server/mock-webhook-store.ts",
        ("fetch(", "axios", "http.request", "https.request"),
    )
    require_all(
        failures,
        "web/src/features/webhook/components/webhook-management.tsx",
        (
            "create.reset()",
            "rotate.reset()",
            "setSecret(null)",
            "webhookTestEvent",
        ),
    )
    require_all(
        failures,
        "web/src/test/webhook-route.test.ts",
        (
            "rejects cross-origin writes before authentication or body parsing",
            "preserves one-time secrets, CAS versions, and secret-free endpoint lists",
            "keeps mock test delivery idempotent, terminal, minimized, and network-free",
            "forwards only fixed paths and reviewed conditional and idempotency headers",
        ),
    )
    require_all(
        failures,
        "web/src/test/webhook-ui.test.tsx",
        (
            "renders endpoint and minimized delivery metadata without exposing secret material",
            "shows a created signing secret once and clears mutation data immediately",
            "queues only a fixed endpoint test with a fresh canonical idempotency key",
        ),
    )
    check_web_routes(failures)

    require_all(
        failures,
        "api/scripts/verify-compose.sh",
        (
            'webhook_flow="skipped"',
            "webhook endpoint list leaked secret material",
            "webhook worker batch failed",
            "webhook ledger contains",
            "webhook schema has ${webhook_query_indexes}/7 query-shaped indexes",
            "webhook migration rollback failed",
            "webhook migration re-apply created",
        ),
    )
    require_all(
        failures,
        ".github/workflows/container.yml",
        (
            "Verify webhook delivery Compose lifecycle",
            "OPTIONAL_STARTERS: organization,webhook",
        ),
    )
    require_all(
        failures,
        "api/docs/WEBHOOKS.md",
        ("Production Configuration", "Operations", "SSRF", "Replacement And Removal"),
    )
    require_all(
        failures,
        "web/docs/WEBHOOKS.md",
        ("Browser Surface", "Production", "Development Mock", "Downstream Removal"),
    )

    if failures:
        print("Webhook boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print("Webhook boundary check passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
