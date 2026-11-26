# Third-party skills

Copied (not linked) so they can't change under us; review before updating. Eval data and template
assets were left out.

| Skills | Source | Commit | License |
| --- | --- | --- | --- |
| golang-code-style, golang-concurrency, golang-context, golang-database, golang-error-handling, golang-naming, golang-refactoring, golang-safety, golang-security, golang-structs-interfaces, golang-testing | https://github.com/samber/cc-skills-golang | d68d80a (2026-11-08) | MIT (LICENSE-samber-cc-skills-golang) |
| svelte-core-bestpractices | https://github.com/sveltejs/ai-tools (packages/opencode/skills) | f9d22f2 (2026-11-17) | MIT (LICENSE-sveltejs-ai-tools) |

Project decisions that override these skills:
- Work sequentially; don't fan out sub-agents ("Orchestration mode" / ultracode) unless asked.
- Use the standard library; don't add samber/* helper libraries (oops, lo, do, ...).
- Database migrations stay in backend/internal/repository/migrations (versioned runner), not an external tool.
- golang-project-layout was not installed (it writes to CLAUDE.md without asking); the layout already follows cmd/ + internal/.
