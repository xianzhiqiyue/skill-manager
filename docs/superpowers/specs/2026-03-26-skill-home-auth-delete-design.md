# Skill Home Registry Auth And Delete Design

## 1. Goal

Make `skill-home-cli` and the bundled `@skill-home/skill-home-manager` workflow support deleting remotely published skills owned by the current user, while keeping registry access rules simple:

- publishing and deleting require login
- pulling, installing, searching, inspecting, and updating public skills do not require login
- private skills remain protected by server-side authorization

This is an authorization and command-surface cleanup, not a registry API redesign.

## 2. Current State

### 2.1 Server behavior already exists

The registry server already enforces the core ownership and auth rules:

- creating skills requires authentication
- publishing new versions requires authentication
- deleting a skill requires authentication and ownership
- deleting a version requires authentication and ownership
- reading public skills is anonymous
- reading private skills returns `403` unless the caller is the owner

Relevant handlers:

- `skill-home-server/internal/api/handlers/skill.go`
- `skill-home-server/internal/api/handlers/version.go`

### 2.2 CLI behavior is incomplete

The CLI already requires login for `push`, but it does not expose first-class commands for deleting a remote skill or deleting a remote version.

The registry client already has low-level methods:

- `DeleteSkill(namespace, name)`
- `DeleteVersion(namespace, name, version)`

But end users cannot invoke them from the command line.

### 2.3 Skill workflow is incomplete

`skills/skill-home-manager/SKILL.md` currently covers local creation, validation, packaging, syncing, install, and environment diagnosis, but it does not include remote deletion workflows for published registry entries.

## 3. User-Facing Outcome

After this change, Skill Home should behave like this:

- `skill-home push [path]` requires login
- `skill-home delete <@namespace/name>` requires login
- `skill-home delete-version <@namespace/name@version>` requires login
- `skill-home pull/install/update/search/info/list --remote` do not require login for public skills
- attempts to access private skills without permission still fail on the server and are surfaced clearly in the CLI
- `@skill-home/skill-home-manager` explicitly supports remote delete workflows

## 4. Command Design

### 4.1 Recommended command shape

Use explicit destructive commands instead of overloading existing ones:

- `skill-home delete <@namespace/name>`
- `skill-home delete-version <@namespace/name@version>`

This is preferred over a single overloaded `delete` command because:

- it avoids ambiguity between deleting a whole skill and deleting one version
- it avoids confusion with local `uninstall`
- it makes destructive actions harder to trigger accidentally

### 4.2 Input rules

`skill-home delete`:

- accepts only skill-level references
- rejects `@namespace/name@version`
- tells the user to use `delete-version` when a version is present

`skill-home delete-version`:

- requires a versioned skill reference
- rejects `@namespace/name` without a version
- tells the user to include an explicit version

### 4.3 Confirmation model

Both commands should:

- ask for confirmation by default
- support `--yes` to skip confirmation for CI or scripting

Default confirmation text should name the exact remote target being deleted so the user is forced to re-read the target.

## 5. Authorization Contract

### 5.1 Commands that require login

The CLI should enforce login before sending the request for:

- `push`
- `delete`
- `delete-version`
- existing account-scoped commands that already require login, such as `activity`, `rate`, and `whoami`

The shared error message should stay consistent:

`未登录，请先运行 'skill-home login'`

### 5.2 Commands that do not require login

The CLI should not require login up front for:

- `pull`
- `install`
- `update`
- `search`
- `info`
- `list --remote`

These commands should keep working anonymously for public skills.

### 5.3 Private-skill behavior

For private skills:

- the server remains the source of truth
- anonymous calls continue to fail with `403`
- the CLI should surface a clearer message for read operations, for example:
  `访问被拒绝：该 skill 可能是私有的，请先运行 'skill-home login' 并确认你有权限`

This keeps public access simple without weakening private-skill protection.

## 6. CLI Structure

### 6.1 New command files

Add focused command implementations:

- `skill-home-cli/internal/cmd/delete.go`
- `skill-home-cli/internal/cmd/delete_version.go`

Responsibilities:

- parse and validate the remote reference
- require login
- prompt for confirmation unless `--yes`
- call the registry client
- print a clear success message

### 6.2 Shared auth helper

Add a small helper in `skill-home-cli/internal/cmd/registry_helpers.go` for commands that must require login before making registry mutations.

This helper should not be used by public read flows such as `pull`, `install`, or `update`.

### 6.3 Root command registration

Register the new commands in:

- `skill-home-cli/internal/cmd/root.go`

This keeps remote destructive actions visible and first-class instead of hiding them behind internal helpers.

## 7. Skill Workflow Changes

### 7.1 `@skill-home/skill-home-manager`

Update the skill definition so it explicitly covers:

- deleting a remotely published skill owned by the current user
- deleting a remotely published version owned by the current user

### 7.2 Command mapping

Add these mappings to the skill:

- “删除已发布 skill” -> `skill-home delete <@namespace/name>`
- “删除已发布某个版本” -> `skill-home delete-version <@namespace/name@version>`

The skill should also explicitly state:

- `push/delete/delete-version` require login
- `pull/install/update/search/info` do not require login for public skills

### 7.3 Workflow reference updates

Update the skill reference doc to include examples such as:

```bash
skill-home login
skill-home delete @testuser/skill-home-manager
skill-home delete-version @skill-home/skill-home-manager@0.2.0
```

No extra helper script is needed for deletion. Keeping deletion as an explicit CLI command is safer than wrapping it in an opaque script.

## 8. Testing Strategy

### 8.1 CLI tests

Add command-level tests that verify:

- `delete` fails fast when not logged in
- `delete-version` fails fast when not logged in
- `delete` rejects versioned refs
- `delete-version` rejects refs without a version
- successful paths call the correct registry client method

### 8.2 Server contract tests

Add or update integration tests to lock in the desired auth contract:

- unauthenticated publish is rejected
- unauthenticated skill delete is rejected
- unauthenticated version delete is rejected
- public skill detail/download remain anonymously readable
- private skill detail/download are still protected

The server code already mostly behaves this way; the main goal is to codify the contract with tests so CLI assumptions stay safe.

## 9. Risks And Guardrails

Main risks:

- confusing remote delete with local uninstall
- accidentally deleting an entire skill when the user intended to delete only one version
- introducing client-side login requirements that unnecessarily break public read flows

Guardrails:

- separate `delete` and `delete-version`
- explicit confirmation by default
- `--yes` required for non-interactive destructive flows
- keep server-side ownership checks unchanged
- avoid adding login gates to anonymous public read paths

## 10. Out Of Scope

This design does not include:

- changing registry ownership rules
- adding admin delete overrides
- deleting local cached skills through the new delete commands
- redesigning the registry API
- bulk delete operations

## 11. Implementation Direction

The minimal implementation should:

1. add CLI delete commands
2. enforce login only on mutating registry commands
3. improve read-flow error messages for private skills
4. update `@skill-home/skill-home-manager`
5. add tests that lock the auth boundary and delete contract

That delivers the requested behavior without expanding scope into a larger registry refactor.
