#!/usr/bin/env python3

"""Keep the optional notification starter idempotent, lease-safe, private, and aligned."""

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


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "contracts/NOTIFICATIONS.md",
        (
            "OPTIONAL_STARTERS=notification",
            "NEXT_PUBLIC_OPTIONAL_FEATURES=notification",
            "**notification** is one immutable application event",
            "**delivery** is that notification's channel-specific execution record",
            "domain.NotificationPublisher",
            "there is deliberately no",
            "browser or public HTTP endpoint that can create notifications",
            "(user_id, idempotency_key)",
            "required channel is always selected",
            "random lease token",
            "SKIP LOCKED",
            "notification-email-<delivery_id>",
            "GET /v1/notification-status",
            "PUT /v1/notification-read-state",
            "GET/PUT /api/notification-preferences",
            "private, no-store",
            "NOTIFICATION.NOT_FOUND",
            "NOTIFICATION.IDEMPOTENCY_CONFLICT",
            "NOTIFICATION.INVALID_CHANNEL",
            "Deliberate Deferrals",
        ),
    )
    require_all(
        failures,
        "api/internal/domain/notification.go",
        (
            'NotificationChannelInApp NotificationChannel = "in_app"',
            'NotificationChannelEmail NotificationChannel = "email"',
            "type NotificationPublication struct",
            "RequiredChannels []NotificationChannel",
            "type NotificationPublisher interface",
            "type NotificationDispatcher interface",
            'EventNotificationCreated = "notification.created"',
        ),
    )
    require_all(
        failures,
        "api/internal/modules/notification/model.go",
        (
            "idx_notifications_user_idempotency",
            "PublicationHash string",
            "idx_notification_deliveries_notification_channel",
            "idx_notification_deliveries_pending",
            "idx_notification_deliveries_leased",
            "notification_deliveries_channel_check",
            "notification_deliveries_status_check",
            "LeaseToken      string",
            "LeaseExpiresAt  *time.Time",
            "DestinationHash string",
            "LastFailureCode string",
            'return "notification_preferences"',
        ),
    )
    require_all(
        failures,
        "api/internal/modules/notification/repository.go",
        (
            "clause.OnConflict",
            'Columns:   []clause.Column{{Name: "user_id"}, {Name: "idempotency_key"}}',
            "existing.PublicationHash != publicationHash",
            "selectPublicationChannels(publication, preference)",
            "visibleNotifications",
            "inAppVisibilitySubquery",
            'Options: "SKIP LOCKED"',
            'Where("id = ? AND channel = ? AND attempts = ? AND attempts < ?"',
            'Where("id = ? AND status = ? AND lease_token = ?"',
            "destination_hash = ''",
            "verify notification email destination",
            "failureRetryExhausted",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/notification/service.go",
        (
            "maxDeliveryAttempts  = uint8(5)",
            "normalizePublication(publication)",
            "publicationFingerprint(normalized)",
            "result.Created && s.events != nil",
            "NotificationChannelInApp",
            "NotificationChannelEmail",
            "preference.InAppEnabled",
            "preference.EmailEnabled",
            "hashDestination(delivery.RecipientEmail)",
            "failureRouteChanged",
            'fmt.Sprintf("notification-email-%d", delivery.ID)',
            "html.EscapeString(delivery.Body)",
            "deliveryRetryDelay(delivery.Attempts)",
            "auditstarter.RecordChange",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/notification/provider.go",
        (
            '"notification"',
            'WithStarterDependencies("user", "audit")',
            "2026_07_15_020000_create_notification_tables",
            "wire.Bind(new(domain.NotificationPublisher)",
            "wire.Bind(new(domain.NotificationDispatcher)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/notification/routes.go",
        (
            'WithMiddleware("auth")',
            'GET("/notifications"',
            'PATCH("/notifications/:id"',
            'GET("/notification-status"',
            'PUT("/notification-read-state"',
            'GET("/notification-preferences"',
            'PUT("/notification-preferences"',
        ),
    )
    forbid_any(
        failures,
        "api/internal/modules/notification/routes.go",
        ('POST("/notifications"',),
    )
    require_all(
        failures,
        "api/database/migrations/2026_07_15_020000_create_notification_tables.go",
        (
            "UseTransaction: true",
            "notification.NotificationPO{}",
            "notification.NotificationDeliveryPO{}",
            "notification.NotificationPreferencePO{}",
            "DropTable(&notification.NotificationDeliveryPO{})",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/console/commands/notification.go",
        (
            'return "notification:work"',
            "--batch=25",
            "--poll=2s",
            "--max-attempts=0",
            "signal.NotifyContext",
            "NotificationDispatcher",
            "cfg.Once || (cfg.MaxAttempts > 0",
            "errorDelay = min(errorDelay*2, 30*time.Second)",
        ),
    )
    forbid_any(
        failures,
        "api/internal/infra/console/commands/notification.go",
        ("--max-jobs", "MaxJobs"),
    )
    require_all(
        failures,
        "api/internal/infra/console/commands/manifest.go",
        ("NewNotificationWorkCommand()",),
    )
    require_all(
        failures,
        "api/internal/infra/email/email.go",
        (
            "SendEmailIdempotent(",
            'request.Header.Set("Idempotency-Key", idempotencyKey)',
            "len(idempotencyKey) > 256",
            "containsControl(idempotencyKey)",
        ),
    )
    require_all(
        failures,
        "api/internal/domain/error_codes.go",
        (
            '"NOTIFICATION.NOT_FOUND"',
            '"NOTIFICATION.IDEMPOTENCY_CONFLICT"',
            '"NOTIFICATION.INVALID_CHANNEL"',
        ),
    )
    require_all(
        failures,
        "web/src/config/optional-features.ts",
        ("'notification'",),
    )
    require_all(
        failures,
        "web/src/features/notification/schemas.ts",
        (
            "from 'zod/mini'",
            "notificationPageEnvelopeSchema = strictObject",
            "notificationPreferenceSchema = strictObject",
            "decodeURIComponent(parsed.pathname)",
            "!decodedPath.includes('\\\\')",
            "containsControl(decodedPath)",
            "maximum(Number.MAX_SAFE_INTEGER)",
        ),
    )
    require_all(
        failures,
        "web/src/features/notification/services/notification-service.ts",
        (
            "notificationPageEnvelopeSchema.safeParse",
            "notificationSchema.safeParse",
            "notificationStatusSchema.safeParse",
            "notificationPreferenceSchema.safeParse",
            "ClientErrorCode.INVALID_RESPONSE",
        ),
    )
    require_all(
        failures,
        "web/src/features/notification/server/notification-route.ts",
        (
            "resolveNotificationRoute(",
            "isWebFeatureEnabled('notification')",
            "guardSameOriginMutation(request)",
            "readJsonBody(request, maxMutationBodyBytes)",
            "privateNoStoreResponse(response, ['Cookie'])",
            "path: 'notification-status'",
            "path: 'notification-read-state'",
            "path: 'notification-preferences'",
            "NOTIFICATION_NOT_FOUND",
        ),
    )
    require_all(
        failures,
        "web/src/features/notification/server/mock-notification-store.ts",
        (
            "const states = new Map<string, MockNotificationState>()",
            "stateFor(user)",
            "item.id <= throughId",
            "unread_count:",
            "replacePreference(",
        ),
    )
    require_all(
        failures,
        "web/src/features/notification/hooks/use-notifications.ts",
        (
            "refetchInterval: 60_000",
            "useNotifications(enabled: boolean)",
            "useNotificationPreference(enabled: boolean)",
            "LOCAL_ERROR_HANDLING_META",
        ),
    )
    require_all(
        failures,
        "web/src/features/notification/components/notification-center.tsx",
        (
            "highestLoadedId",
            "markRead.mutateAsync(highestLoadedId)",
            "router.push(item.action_url)",
            "aria-label={label}",
        ),
    )
    forbid_any(
        failures,
        "web/src/features/notification/components/notification-center.tsx",
        ("dangerouslySetInnerHTML",),
    )
    for route in (
        "web/src/app/api/notifications/route.ts",
        "web/src/app/api/notifications/[notificationId]/route.ts",
        "web/src/app/api/notification-status/route.ts",
        "web/src/app/api/notification-read-state/route.ts",
        "web/src/app/api/notification-preferences/route.ts",
    ):
        require_all(failures, route, ("privateNotificationResponse",))

    require_all(
        failures,
        "api/docs/NOTIFICATIONS.md",
        (
            "luas notification:work",
            "same image, database",
            "retention",
            "To remove the starter",
        ),
    )
    require_all(
        failures,
        "web/docs/NOTIFICATIONS.md",
        (
            "NEXT_PUBLIC_OPTIONAL_FEATURES=notification",
            "strict Zod schemas",
            "To remove it",
        ),
    )
    require_all(
        failures,
        "api/scripts/verify-compose.sh",
        (
            'notification_flow="skipped"',
            'notification_migration_flow="skipped"',
            "/v1/notification-status",
            "/v1/notification-read-state",
            "/v1/notification-preferences",
            "notification migration re-apply created",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/notification/service_test.go",
        (
            "TestPublishIsIdempotentAndRejectsConflictingReplay",
            "TestNotificationCenterEnforcesOwnershipAndReadHighWaterMark",
            "TestDispatchRetriesWithStableProviderIdempotencyKey",
            "TestExpiredLeaseCannotBeCompletedByPreviousWorker",
            "TestBindEmailDestinationIsIdempotentOnlyForTheCurrentLeaseAndRoute",
            "TestPublicationValidationRejectsUnsafeBoundaryValues",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/console/commands/notification_test.go",
        ("TestRunNotificationWorkerOnceReturnsDispatchErrorAndCompletedCount",),
    )
    require_all(
        failures,
        "web/src/test/notification-route.test.ts",
        (
            "does not reveal another mock user notification",
            "rejects cross-origin writes before authentication or body parsing",
            "forwards only fixed production notification paths",
        ),
    )

    old_manager = ROOT / "api/internal/infra/notification/notification.go"
    if old_manager.exists():
        failures.append(
            "api/internal/infra/notification/notification.go must stay removed; the route-owning starter owns notification workflow"
        )

    if failures:
        print("Notification boundary check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print("Notification boundary check passed (idempotent, lease-safe, private).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
