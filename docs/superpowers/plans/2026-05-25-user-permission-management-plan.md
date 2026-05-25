# User Permission Management Plan

Date: 2026-05-25

## Goal

Ship the first productized user-permission management slice for Skill Home without replacing the current `is_admin` / `is_super_admin` model.

## Scope

- Extend super-admin user listing with pagination, text search, role filter, and status filter.
- Add derived `role` in admin user responses.
- Add backend guardrails for dangerous user updates:
  - reject self deactivation;
  - reject removing the last super admin;
  - keep all user updates audited.
- Add Web user management page under Settings for super admins.
- Add docs for the current role matrix and management endpoints.

## Non-goals

- No generic RBAC tables in this slice.
- No API Key scope model in this slice.
- No organization/team-level permission model in this slice.

## Acceptance

- Super admin can list, search, filter, enable/disable, change admin/super-admin flags, and reset a user's password from Web.
- Non-super-admin users cannot reach user management APIs or UI entry points.
- The API refuses self deactivation and refuses to remove the last super admin.
- Backend and Web tests cover the new behavior.
- README/API docs describe the permission matrix and endpoints.

## Verification

- `cd skill-home-server && go test ./...`
- `cd skill-home-web && npm test`
- `cd skill-home-web && npm run build`

## Completion Notes

- Backend user management, profile/password self-service, and global audit APIs are implemented.
- Web Settings now includes a super-admin-only Users page.
- Permission matrix and endpoint docs are updated in README/API/requirements docs.
- API Key scope and full RBAC remain out of scope for this slice.
