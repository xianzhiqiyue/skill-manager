# Skill Home Registry Auth And Delete Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add first-class CLI commands for deleting remotely published skills and versions, keep publish/delete behind login, preserve anonymous public read flows, and extend `@skill-home/skill-home-manager` to cover remote deletion.

**Architecture:** Treat the registry server as the source of truth for authorization and ownership, then make the CLI align with that contract instead of re-inventing it. First lock the existing server auth behavior with characterization tests, then add CLI mutation commands and shared auth/error helpers, and finally update the bundled skill and docs so the new workflow is discoverable and safe.

**Tech Stack:** Go, Cobra, Viper, Gin, GORM, Vitest-free Go tests, Markdown docs, Skill Home skill metadata

---

## File Structure Map

### Existing files to keep and modify

- Modify: `skill-home-server/internal/api/handlers/handler_integration_test.go`
  - Lock publish/delete auth requirements and anonymous public/private read behavior.
- Modify: `skill-home-cli/internal/cmd/root.go`
  - Register new destructive remote commands.
- Modify: `skill-home-cli/internal/cmd/registry_helpers.go`
  - Add shared login enforcement and read-error wrapping helpers.
- Modify: `skill-home-cli/internal/cmd/info.go`
  - Surface clearer anonymous/private access errors.
- Modify: `skill-home-cli/internal/cmd/pull.go`
  - Keep anonymous public pulls working while improving private-access error messaging.
- Modify: `skill-home-cli/internal/cmd/update.go`
  - Reuse the read-error wrapper while continuing to skip inaccessible skills instead of aborting the whole update.
- Modify: `skill-home-cli/internal/cmd/push_test.go`
  - Keep help-smoke tests stable if shared root registration changes affect flag wiring.
- Modify: `skill-home-cli/internal/registry/client_test.go`
  - Add regression coverage for `DELETE` requests used by the new commands.
- Modify: `skill-home-cli/README.md`
  - Document `delete` and `delete-version`, plus the auth boundary.
- Modify: `README.md`
  - Keep top-level auth expectations aligned with the CLI.
- Modify: `skills/skill-home-manager/SKILL.md`
  - Extend the workflow to support remote deletion.
- Modify: `skills/skill-home-manager/references/cli-workflows.md`
  - Add concrete delete examples and auth expectations.

### New files to create

- Create: `skill-home-cli/internal/cmd/delete.go`
  - Implement `skill-home delete <@namespace/name>`.
- Create: `skill-home-cli/internal/cmd/delete_version.go`
  - Implement `skill-home delete-version <@namespace/name@version>`.
- Create: `skill-home-cli/internal/cmd/delete_test.go`
  - Test login enforcement, ref validation, confirmation bypass, and command registration.
- Create: `skill-home-cli/internal/cmd/registry_helpers_test.go`
  - Test shared auth and private-read error helpers.

### Optional files to touch only if characterization tests expose a gap

- Modify: `skill-home-server/cmd/server/main.go`
  - Only if route middleware does not actually match the desired auth contract.
- Modify: `skill-home-server/internal/api/handlers/skill.go`
  - Only if a private/public read contract test exposes a real mismatch.
- Modify: `skill-home-server/internal/api/handlers/version.go`
  - Only if a delete/download auth test exposes a real mismatch.

---

### Task 1: Lock the server auth contract before changing CLI behavior

**Files:**
- Modify: `skill-home-server/internal/api/handlers/handler_integration_test.go`
- Modify: `skill-home-server/cmd/server/main.go` only if the tests expose a real route gap
- Modify: `skill-home-server/internal/api/handlers/skill.go` only if the tests expose a real handler gap
- Modify: `skill-home-server/internal/api/handlers/version.go` only if the tests expose a real handler gap

- [ ] **Step 1: Write characterization tests for the approved auth boundary**

```go
func TestCreateSkillRequiresAuth(t *testing.T) {
	db := newTestDatabase(t)
	router := gin.New()
	router.POST("/api/v1/skills", CreateSkill(db, newTestObjectStorage(t), validator.NewScanner()))

	req, _ := newCreateSkillRequest(t, map[string]string{
		"namespace": "team",
		"name": "github",
		"version": "1.0.0",
	}, newSkillArchive(t))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPublicSkillDetailAllowsAnonymousAccess(t *testing.T) {
	db := newTestDatabase(t)
	seedPublicSkill(t, db, "team", "github")

	router := gin.New()
	router.GET("/api/v1/skills/:namespace/:name", middleware.OptionalAuth(db), GetSkill(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/team/github", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}
```

- [ ] **Step 2: Run the characterization tests**

Run: `cd skill-home-server && go test ./internal/api/handlers -run 'TestCreateSkillRequiresAuth|TestDeleteSkillRequiresAuth|TestDeleteVersionRequiresAuth|TestPublicSkillDetailAllowsAnonymousAccess|TestPrivateSkillDetailRejectsAnonymousAccess|TestPublicDownloadAllowsAnonymousAccess|TestPrivateDownloadRejectsAnonymousAccess'`

Expected:
- PASS if the current server already matches the approved contract
- FAIL only if there is a real mismatch between handlers/routes and the approved behavior

- [ ] **Step 3: Fix only the smallest server gap if any test fails**

```go
api.GET("/skills/:namespace/:name", middleware.OptionalAuth(db), handlers.GetSkill(db))
api.GET("/skills/:namespace/:name/versions", middleware.OptionalAuth(db), handlers.ListVersions(db))
api.GET("/download/:namespace/:name/:version", middleware.OptionalAuth(db), middleware.RateLimit(), handlers.DownloadSkill(db, objStorage))

auth := api.Group("/")
auth.Use(middleware.Auth(db))
auth.POST("/skills", handlers.CreateSkill(db, objStorage, scanner))
auth.DELETE("/skills/:namespace/:name", handlers.DeleteSkill(db, objStorage))
auth.DELETE("/skills/:namespace/:name/versions/:version", handlers.DeleteVersion(db, objStorage))
```

- [ ] **Step 4: Re-run the server contract tests**

Run: `cd skill-home-server && go test ./internal/api/handlers -run 'TestCreateSkillRequiresAuth|TestDeleteSkillRequiresAuth|TestDeleteVersionRequiresAuth|TestPublicSkillDetailAllowsAnonymousAccess|TestPrivateSkillDetailRejectsAnonymousAccess|TestPublicDownloadAllowsAnonymousAccess|TestPrivateDownloadRejectsAnonymousAccess'`

Expected: PASS

- [ ] **Step 5: Commit the characterization contract**

```bash
git add skill-home-server/internal/api/handlers/handler_integration_test.go skill-home-server/cmd/server/main.go skill-home-server/internal/api/handlers/skill.go skill-home-server/internal/api/handlers/version.go
git commit -m "test: lock registry auth contract"
```

### Task 2: Add explicit remote delete commands to the CLI

**Files:**
- Create: `skill-home-cli/internal/cmd/delete.go`
- Create: `skill-home-cli/internal/cmd/delete_version.go`
- Create: `skill-home-cli/internal/cmd/delete_test.go`
- Modify: `skill-home-cli/internal/cmd/root.go`
- Modify: `skill-home-cli/internal/cmd/registry_helpers.go`
- Modify: `skill-home-cli/internal/registry/client_test.go`
- Modify: `skill-home-cli/internal/cmd/push_test.go` only if root help smoke coverage needs an update

- [ ] **Step 1: Write the failing command tests first**

```go
func TestRunDeleteRequiresLogin(t *testing.T) {
	t.Setenv("SKILL_HOME_API_KEY", "")
	viper.Set("registry.api_key", "")

	err := runDelete("@team/reviewer", &deleteOptions{yes: true})
	if err == nil || !strings.Contains(err.Error(), "未登录，请先运行 'skill-home login'") {
		t.Fatalf("expected login error, got %v", err)
	}
}

func TestRunDeleteRejectsVersionedRef(t *testing.T) {
	viper.Set("registry.api_key", "sk_test")

	err := runDelete("@team/reviewer@1.0.0", &deleteOptions{yes: true})
	if err == nil || !strings.Contains(err.Error(), "delete-version") {
		t.Fatalf("expected version guidance error, got %v", err)
	}
}

func TestRunDeleteVersionRequiresExplicitVersion(t *testing.T) {
	viper.Set("registry.api_key", "sk_test")

	err := runDeleteVersion("@team/reviewer", &deleteVersionOptions{yes: true})
	if err == nil || !strings.Contains(err.Error(), "必须包含版本号") {
		t.Fatalf("expected version-required error, got %v", err)
	}
}
```

- [ ] **Step 2: Run the focused CLI tests to verify they fail**

Run: `cd skill-home-cli && go test ./internal/cmd -run 'TestRunDelete|TestRunDeleteVersion|TestDeleteHelpDoesNotPanic'`

Expected: FAIL because the delete commands and shared helpers do not exist yet.

- [ ] **Step 3: Implement the minimal shared auth and mutation-client hooks**

```go
type registryDeleteClient interface {
	DeleteSkill(namespace, name string) error
	DeleteVersion(namespace, name, version string) error
}

var newRegistryDeleteClient = func() registryDeleteClient {
	return newRegistryClient()
}

func requireRegistryLogin() error {
	if strings.TrimSpace(viper.GetString("registry.api_key")) == "" {
		return fmt.Errorf("未登录，请先运行 'skill-home login'")
	}
	return nil
}
```

- [ ] **Step 4: Implement `delete` and `delete-version` with confirmation-by-default**

```go
func runDelete(skillRef string, opts *deleteOptions) error {
	if err := requireRegistryLogin(); err != nil {
		return err
	}

	namespace, name, version, err := config.ParseSkillRef(skillRef)
	if err != nil {
		return err
	}
	if version != "" {
		return fmt.Errorf("skill-home delete 不接受版本号，请改用 'skill-home delete-version %s'", skillRef)
	}
	if !opts.yes && !confirmDelete(fmt.Sprintf("@%s/%s", namespace, name)) {
		return fmt.Errorf("操作已取消")
	}

	if err := newRegistryDeleteClient().DeleteSkill(namespace, name); err != nil {
		return fmt.Errorf("删除远程 skill 失败: %w", err)
	}

	fmt.Printf("%s 已删除 @%s/%s\n", color.GreenString("✓"), namespace, name)
	return nil
}
```

- [ ] **Step 5: Add low-level client regression tests for DELETE requests**

```go
func TestDeleteSkillSendsDeleteAndAuthorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/skills/team/reviewer" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "sk_test")
	if err := client.DeleteSkill("team", "reviewer"); err != nil {
		t.Fatalf("DeleteSkill returned error: %v", err)
	}
}
```

- [ ] **Step 6: Run the CLI and registry tests**

Run: `cd skill-home-cli && go test ./internal/cmd ./internal/registry -run 'TestRunDelete|TestRunDeleteVersion|TestDeleteHelpDoesNotPanic|TestDeleteVersionHelpDoesNotPanic|TestDeleteSkillSendsDeleteAndAuthorization|TestDeleteVersionSendsDeleteAndAuthorization'`

Expected: PASS

- [ ] **Step 7: Commit the new command surface**

```bash
git add skill-home-cli/internal/cmd/delete.go skill-home-cli/internal/cmd/delete_version.go skill-home-cli/internal/cmd/delete_test.go skill-home-cli/internal/cmd/root.go skill-home-cli/internal/cmd/registry_helpers.go skill-home-cli/internal/registry/client_test.go skill-home-cli/internal/cmd/push_test.go
git commit -m "feat: add registry delete commands"
```

### Task 3: Improve anonymous read errors without breaking public pulls and updates

**Files:**
- Create: `skill-home-cli/internal/cmd/registry_helpers_test.go`
- Modify: `skill-home-cli/internal/cmd/registry_helpers.go`
- Modify: `skill-home-cli/internal/cmd/info.go`
- Modify: `skill-home-cli/internal/cmd/pull.go`
- Modify: `skill-home-cli/internal/cmd/update.go`

- [ ] **Step 1: Write the failing helper tests for private-read messaging**

```go
func TestWrapRegistryReadErrorSuggestsLoginForForbidden(t *testing.T) {
	err := wrapRegistryReadError("获取技能详情", &registry.APIError{
		Code:    "FORBIDDEN",
		Message: "Access denied",
	})

	if err == nil || !strings.Contains(err.Error(), "请先运行 'skill-home login'") {
		t.Fatalf("expected private-skill guidance, got %v", err)
	}
}

func TestWrapRegistryReadErrorLeavesOtherErrorsAlone(t *testing.T) {
	err := wrapRegistryReadError("下载失败", io.EOF)
	if err == nil || !strings.Contains(err.Error(), "EOF") {
		t.Fatalf("expected wrapped original error, got %v", err)
	}
}
```

- [ ] **Step 2: Run the focused helper tests to verify they fail**

Run: `cd skill-home-cli && go test ./internal/cmd -run 'TestWrapRegistryReadError'`

Expected: FAIL because the helper does not exist yet.

- [ ] **Step 3: Implement the read-error wrapper and wire it into public read commands**

```go
func wrapRegistryReadError(action string, err error) error {
	var apiErr *registry.APIError
	if errors.As(err, &apiErr) && apiErr.Code == "FORBIDDEN" {
		return fmt.Errorf("%s: 访问被拒绝：该 skill 可能是私有的，请先运行 'skill-home login' 并确认你有权限", action)
	}
	return fmt.Errorf("%s: %w", action, err)
}
```

```go
skill, err := client.GetSkill(namespace, name)
if err != nil {
	return wrapRegistryReadError("获取技能详情失败", err)
}
```

```go
if err := client.Download(namespace, name, version, tmpFile); err != nil {
	return nil, wrapRegistryReadError("下载失败", err)
}
```

- [ ] **Step 4: Verify update still skips inaccessible skills instead of aborting the whole run**

Run: `cd skill-home-cli && go test ./internal/cmd -run 'TestWrapRegistryReadError|TestRunDelete|TestRunDeleteVersion'`

Expected: PASS for helper and delete command tests; `update` keeps its continue-on-error behavior because only the message wrapper changed.

- [ ] **Step 5: Commit the read-flow polish**

```bash
git add skill-home-cli/internal/cmd/registry_helpers_test.go skill-home-cli/internal/cmd/registry_helpers.go skill-home-cli/internal/cmd/info.go skill-home-cli/internal/cmd/pull.go skill-home-cli/internal/cmd/update.go
git commit -m "feat: clarify private registry read errors"
```

### Task 4: Extend `@skill-home/skill-home-manager` and the docs for remote deletion

**Files:**
- Modify: `skills/skill-home-manager/SKILL.md`
- Modify: `skills/skill-home-manager/references/cli-workflows.md`
- Modify: `skill-home-cli/README.md`
- Modify: `README.md`

- [ ] **Step 1: Update the skill definition to include remote deletion workflows**

```md
## 触发场景

- 用户要删除自己已发布的远端 skill
- 用户要删除自己已发布的远端 skill 版本

## 命令选择规则

- 删除远端 skill: `skill-home delete`
- 删除远端版本: `skill-home delete-version`

## 用户请求到动作的映射

- “帮我删除已发布 skill”: `skill-home delete <@namespace/name>`
- “帮我删除已发布某个版本”: `skill-home delete-version <@namespace/name@version>`
```

- [ ] **Step 2: Update CLI docs to make the auth boundary explicit**

```md
| `skill-home push [path]` | 发布 skill 到注册中心（需要登录） |
| `skill-home delete <skill-ref>` | 删除远端 skill（需要登录） |
| `skill-home delete-version <skill-ref>` | 删除远端指定版本（需要登录） |

公开 skill 的 `pull/install/update/search/info` 不需要登录；私有 skill 仍需具备权限。
```

- [ ] **Step 3: Verify the docs mention the new commands and auth rules consistently**

Run: `rg -n "delete-version|skill-home delete|需要登录|不需要登录" skills/skill-home-manager/SKILL.md skills/skill-home-manager/references/cli-workflows.md skill-home-cli/README.md README.md`

Expected: all four docs mention the new delete commands and the publish/delete vs. pull/install/update auth boundary.

- [ ] **Step 4: Commit the workflow and docs update**

```bash
git add skills/skill-home-manager/SKILL.md skills/skill-home-manager/references/cli-workflows.md skill-home-cli/README.md README.md
git commit -m "docs: document registry delete workflow"
```

### Task 5: Run full verification before claiming completion

**Files:**
- Modify: none

- [ ] **Step 1: Run all CLI tests**

Run: `cd skill-home-cli && go test ./...`

Expected: PASS

- [ ] **Step 2: Run the server handler tests**

Run: `cd skill-home-server && go test ./internal/api/handlers`

Expected: PASS

- [ ] **Step 3: Smoke-check the new CLI help output**

Run: `cd skill-home-cli && go run ./cmd/skill-home --help && go run ./cmd/skill-home delete --help && go run ./cmd/skill-home delete-version --help`

Expected:
- root help lists `delete` and `delete-version`
- delete help shows a skill-only ref
- delete-version help shows a versioned ref

- [ ] **Step 4: Record any intentional follow-up gaps**

Expected:
- no server API shape changes beyond locked tests
- no bulk delete support
- no admin override flows
