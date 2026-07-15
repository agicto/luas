#!/usr/bin/env python3

"""Keep optional starter runtime, schema, contracts, and guidance aligned."""

from __future__ import annotations

import re
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


def between(content: str, start: str, end: str) -> str:
    if start not in content or end not in content:
        return ""
    return content.split(start, 1)[1].split(end, 1)[0]


def main() -> int:
    failures: list[str] = []

    require_all(
        failures,
        "api/internal/infra/config/config.go",
        (
            "type StarterConfig struct",
            'env.GetSlice("OPTIONAL_STARTERS", []string{})',
            "DefaultOrganizationInvitationTTL",
            'env.GetDuration("ORGANIZATION_INVITATION_TTL", DefaultOrganizationInvitationTTL)',
        ),
    )
    require_all(
        failures,
        "api/internal/starter/catalog.go",
        (
            "canonical lowercase name",
            "duplicate optional starter",
            "unknown optional starter",
            "is a default starter",
        ),
    )
    require_all(
        failures,
        "api/internal/starter/defaults.go",
        (
            "func NewConfiguredRegistry(",
            "func OptionalManifests(",
            "func ConfiguredMigrations(",
            "func ConfiguredSeeders(",
            "migrator.RegisterMany(registry.Migrations())",
        ),
    )
    require_all(
        failures,
        "api/internal/starter/assembly/starter_manifest.go",
        ("isNilModule", "reflect.ValueOf(module)"),
    )
    require_all(
        failures,
        "api/internal/starter/registry.go",
        (
            "migrationOwners",
            "seederOwners",
            "Activation happens only after every manifest contribution passes preflight.",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/console/commands/migrate.go",
        ("starter.ConfiguredMigrations(cfg)",),
    )
    require_all(
        failures,
        "api/internal/infra/console/commands/db.go",
        ("bootstrap.RunConfiguredSeeders(db, cfg)",),
    )

    try:
        defaults = read("api/internal/starter/defaults.go")
    except FileNotFoundError:
        defaults = ""
    default_segment = between(
        defaults, "func DefaultManifests(", "// DefaultMigrations"
    )
    optional_segment = between(
        defaults, "func OptionalManifests(", "// ConfiguredManifests"
    )
    optional_packages = re.findall(
        r"([a-z][a-z0-9_]*)\.NewStarterManifest", optional_segment
    )
    if optional_packages != ["organization"]:
        failures.append(
            "optional starter catalog must currently contain exactly organization"
        )
    if "organization.NewStarterManifest" in default_segment:
        failures.append("organization must not be part of DefaultManifests")

    for package_name in optional_packages:
        provider_path = f"api/internal/modules/{package_name}/provider.go"
        try:
            provider = read(provider_path)
        except FileNotFoundError:
            failures.append(f"{provider_path} is missing")
            continue
        if f'"{package_name}"' not in provider:
            failures.append(
                f"{provider_path} must use the canonical starter name {package_name!r}"
            )
        migration_names = re.findall(
            r'"([0-9]{4}_[0-9]{2}_[0-9]{2}_[0-9]{6}_[a-z0-9_]+)"',
            provider,
        )
        if not migration_names:
            failures.append(f"{provider_path} must register at least one migration")
        for migration_name in migration_names:
            migration_path = (
                ROOT / "api" / "database" / "migrations" / f"{migration_name}.go"
            )
            if not migration_path.exists():
                failures.append(
                    f"{provider_path} references missing migration {migration_name}"
                )

    require_all(
        failures,
        "contracts/ORGANIZATIONS.md",
        (
            "OPTIONAL_STARTERS=organization",
            "POST /v1/organizations/:id/invitations",
            "POST /v1/organization-invitations/accept",
            "GET /v1/organizations/:id/members",
            "PATCH /v1/organizations/:id/members/:member_id",
            "DELETE /v1/organizations/:id/members/:member_id",
            "POST /v1/organizations/:id/ownership-transfer",
            "ORGANIZATION_INVITATION_TTL",
            "stores only their SHA-256 hash",
            "ORGANIZATION.NOT_FOUND",
            "ORGANIZATION.SLUG_ALREADY_EXISTS",
            "ORGANIZATION.OWNERSHIP_TRANSFER_REQUIRED",
            "ORGANIZATION.INVITATION.INVALID",
            "ORGANIZATION.INVITATION.EXPIRED",
            "ORGANIZATION.INVITATION.EMAIL_MISMATCH",
            "ORGANIZATION.INVITATION.ALREADY_PENDING",
            "ORGANIZATION.MEMBER_ALREADY_EXISTS",
            "ORGANIZATION.MEMBER_NOT_FOUND",
            "ORGANIZATION.MEMBERSHIP_EXIT_REQUIRED",
            "ORGANIZATION.OWNERSHIP_TRANSFER_TARGET_INVALID",
            "Deliberate Deferrals",
        ),
    )
    require_all(
        failures,
        "api/internal/domain/error_codes.go",
        (
            '"ORGANIZATION.NOT_FOUND"',
            '"ORGANIZATION.SLUG_ALREADY_EXISTS"',
            '"ORGANIZATION.OWNERSHIP_TRANSFER_REQUIRED"',
            '"ORGANIZATION.OWNERSHIP_TRANSFER_TARGET_INVALID"',
            '"ORGANIZATION.MEMBERSHIP_EXIT_REQUIRED"',
            '"ORGANIZATION.MEMBER_NOT_FOUND"',
            '"ORGANIZATION.INVITATION.INVALID"',
            '"ORGANIZATION.INVITATION.EXPIRED"',
            '"ORGANIZATION.INVITATION.EMAIL_MISMATCH"',
            '"ORGANIZATION.INVITATION.ALREADY_PENDING"',
            '"ORGANIZATION.MEMBER_ALREADY_EXISTS"',
        ),
    )
    require_all(
        failures,
        "api/internal/bootstrap/domain_error_mappings.go",
        (
            "ErrOrganizationInvitationInvalid, http.StatusNotFound",
            "ErrOrganizationMemberNotFound, http.StatusNotFound",
            "ErrOrganizationInvitationEmailMismatch, http.StatusForbidden",
            "ErrOrganizationOwnershipTransferTargetInvalid, http.StatusConflict",
            "ErrOrganizationMembershipExitRequired, http.StatusConflict",
            "ErrOrganizationInvitationAlreadyPending, http.StatusConflict",
            "ErrOrganizationMemberAlreadyExists, http.StatusConflict",
            "ErrOrganizationInvitationExpired, http.StatusGone",
        ),
    )
    require_all(
        failures,
        "api/database/migrations/2026_07_14_000000_create_organizations_tables.go",
        (
            "UseTransaction: true",
            "organization.OrganizationPO{}",
            "organization.OrganizationMembershipPO{}",
        ),
    )
    require_all(
        failures,
        "api/database/migrations/2026_07_15_000000_create_organization_invitations_table.go",
        (
            "UseTransaction: true",
            "organization.OrganizationInvitationPO{}",
            "DropTable(&organization.OrganizationInvitationPO{})",
        ),
    )
    require_all(
        failures,
        "api/internal/domain/organization.go",
        (
            'TokenHash      string           `json:"-"`',
            "type OrganizationInvitationRepository interface",
            "OrganizationInvitationStatusExpired",
            "ListMembers(ctx context.Context",
            "ChangeMemberRole(ctx context.Context",
            "RemoveMember(ctx context.Context",
            "TransferOwnership(ctx context.Context",
            "CountMembershipsForUser(ctx context.Context",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/organization/model.go",
        (
            'TokenHash      string     `gorm:"size:64;not null;uniqueIndex"`',
            'PendingKey     *string    `gorm:"size:64;uniqueIndex"`',
        ),
    )
    require_all(
        failures,
        "api/internal/modules/organization/routes.go",
        (
            'POST("/organizations/:id/invitations"',
            'GET("/organizations/:id/invitations"',
            'DELETE("/organizations/:id/invitations/:invitation_id"',
            'POST("/organization-invitations/accept"',
            'GET("/organizations/:id/members"',
            'PATCH("/organizations/:id/members/:member_id"',
            'DELETE("/organizations/:id/members/:member_id"',
            'POST("/organizations/:id/ownership-transfer"',
        ),
    )

    require_all(
        failures,
        "api/internal/modules/organization/membership_repository.go",
        (
            "func (r *repository) ListMembers(",
            "func (r *repository) ChangeMemberRole(",
            "func (r *repository) RemoveMember(",
            "func (r *repository) TransferOwnership(",
            'clause.Locking{Strength: "UPDATE"}',
            "CountMembershipsForUser",
        ),
    )
    require_all(
        failures,
        "api/internal/infra/database/transaction_context.go",
        (
            "func ContextWithTransaction(",
            "func TransactionFromContext(",
            "func ResolveContextDB(",
            "must not escape the transaction callback",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/user/repository.go",
        (
            "func (r *repository) DeleteAccount(",
            'clause.Locking{Strength: "UPDATE"}',
            "infradatabase.ContextWithTransaction(ctx, tx)",
            "check(transactionContext)",
            "tx.WithContext(transactionContext).Delete(&po)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/user/service.go",
        (
            "s.repo.DeleteAccount(ctx, userID",
            "s.deletionPolicy.Check(transactionContext, userID)",
        ),
    )
    require_all(
        failures,
        "api/internal/modules/organization/repository.go",
        ("findUndeletedUserForUpdate(tx, owner.UserID)",),
    )
    require_all(
        failures,
        "api/internal/modules/organization/invitation_repository.go",
        ("findUndeletedUserForUpdate(tx, userID)",),
    )
    try:
        organization_repository = read(
            "api/internal/modules/organization/repository.go"
        )
    except FileNotFoundError:
        organization_repository = ""
    else:
        create_owner = between(
            organization_repository,
            "func (r *repository) CreateWithOwner(",
            "func (r *repository) FindForUser(",
        )
        user_lock_at = create_owner.find("findUndeletedUserForUpdate(")
        organization_create_at = create_owner.find("Create(organizationPO)")
        owner_create_at = create_owner.find("Create(ownerPO)")
        if min(user_lock_at, organization_create_at, owner_create_at) < 0 or not (
            user_lock_at < organization_create_at < owner_create_at
        ):
            failures.append(
                "organization creation must lock the undeleted user row before organization and owner membership writes"
            )

    try:
        invitation_repository = read(
            "api/internal/modules/organization/invitation_repository.go"
        )
    except FileNotFoundError:
        invitation_repository = ""
    else:
        accept_marker = "func (r *repository) AcceptInvitation("
        accept_invitation = (
            invitation_repository.split(accept_marker, 1)[1]
            if accept_marker in invitation_repository
            else ""
        )
        user_lock_at = accept_invitation.find("findUndeletedUserForUpdate(")
        membership_create_at = accept_invitation.find("Create(&membershipPO)")
        if min(user_lock_at, membership_create_at) < 0 or not (
            user_lock_at < membership_create_at
        ):
            failures.append(
                "invitation acceptance must lock the undeleted user row before membership creation"
            )
    try:
        membership_repository = read(
            "api/internal/modules/organization/membership_repository.go"
        )
    except FileNotFoundError:
        membership_repository = ""
    else:
        transfer = between(
            membership_repository,
            "func (r *repository) TransferOwnership(",
            "func (r *repository) CountMembershipsForUser(",
        )
        actor_at = transfer.find("findMembershipForUserUpdate(")
        target_at = transfer.find("findMembershipByIDUpdate(")
        demote_at = transfer.find("previousOwner :=")
        promote_at = transfer.find("newOwner :=")
        if min(actor_at, target_at, demote_at, promote_at) < 0 or not (
            actor_at < target_at < demote_at < promote_at
        ):
            failures.append(
                "ownership transfer must lock actor then target and demote/promote in one ordered transaction"
            )
    require_all(
        failures,
        "api/internal/modules/organization/invitation_mailer.go",
        ("html.EscapeString", "email.ErrNotConfigured"),
    )

    try:
        invitation_service = read(
            "api/internal/modules/organization/invitation_service.go"
        )
    except FileNotFoundError:
        failures.append(
            "api/internal/modules/organization/invitation_service.go is missing"
        )
    else:
        persist_at = invitation_service.find("s.invitationRepo.CreateInvitation(")
        send_at = invitation_service.find("s.invitationMailer.SendInvitation(")
        if persist_at < 0 or send_at < 0 or persist_at > send_at:
            failures.append(
                "organization invitations must persist before attempting email delivery"
            )

    try:
        organization_dto = read("api/internal/modules/organization/dto.go")
    except FileNotFoundError:
        failures.append("api/internal/modules/organization/dto.go is missing")
    else:
        invitation_response = between(
            organization_dto,
            "type OrganizationInvitationResponse struct",
            "// CreateOrganizationInvitationResponse",
        )
        if not invitation_response or "Token" in invitation_response:
            failures.append(
                "OrganizationInvitationResponse must exist without exposing a token field"
            )
        member_response = between(
            organization_dto,
            "type OrganizationMemberResponse struct",
            "// OrganizationOwnershipTransferResponse",
        )
        forbidden_member_fields = ("Email", "Phone", "Status", "Password")
        if not member_response or any(
            field in member_response for field in forbidden_member_fields
        ):
            failures.append(
                "OrganizationMemberResponse must exist without private user profile fields"
            )
    require_all(
        failures,
        "api/.env.example",
        (
            "OPTIONAL_STARTERS=",
            "available: organization",
            "ORGANIZATION_INVITATION_TTL=168h",
        ),
    )
    require_all(
        failures,
        "api/docker-compose.yml",
        (
            "OPTIONAL_STARTERS: ${OPTIONAL_STARTERS:-}",
            "ORGANIZATION_INVITATION_TTL: ${ORGANIZATION_INVITATION_TTL:-168h}",
        ),
    )
    require_all(
        failures,
        "api/scripts/verify-compose.sh",
        (
            "*,organization,*)",
            "/v1/organizations/${organization_id}/invitations",
            "ORGANIZATION.INVITATION.ALREADY_PENDING",
            "email_send_status",
            "replacement invitation",
            "/members",
            "/ownership-transfer",
            "ORGANIZATION.MEMBERSHIP_EXIT_REQUIRED",
            "concurrent ownership transfer",
            "concurrent account deletion",
            "orphaned_memberships",
        ),
    )
    require_all(
        failures,
        "api/docs/CONFIGURATION.md",
        (
            "Optional Starter Activation",
            "OPTIONAL_STARTERS=organization",
            "ORGANIZATION_INVITATION_TTL=168h",
        ),
    )
    require_all(
        failures,
        "docs/STARTER_BUSINESS_ROADMAP.md",
        (
            "`organization` optional starter",
            "Foundation only",
            "ownership, invitation, and member-lifecycle kernels",
        ),
    )
    require_all(
        failures,
        "api/.agents/skills/module-creation/SKILL.md",
        ("OptionalManifests", "OPTIONAL_STARTERS=blog"),
    )
    require_all(
        failures,
        ".github/workflows/skill-self-test.yml",
        ("module: [user, apikey, audit, organization]",),
    )

    if failures:
        print("Starter catalog check failed:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(
        f"Starter catalog check passed ({len(optional_packages)} optional starter)."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
