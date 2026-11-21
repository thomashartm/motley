+++
schema = 1
name = "implement-feature-ticket"
description = "Implement one feature ticket and verify its acceptance criteria"
vars = ["additional_items"]
+++
Implement the feature ticket at {{if .Issue.URL}}{{.Issue.URL}}{{else}}[GitHub issue URL]{{end}}.
{{with .Issue.Title}}
Feature: {{.}}
{{end}}
Read the ticket and the applicable AGENTS.md files before editing. Follow the repository rules. Use the ticket's functional requirements and acceptance criteria to define the scope. Treat linked issues as context unless the ticket explicitly requires work from them.

Inspect the existing implementation and its callers. Make the smallest complete change that delivers the required behavior. If a requirement is ambiguous or completion needs a broader change, explain the specific blocker and ask before expanding scope.

If you discover an unrelated problem, check for an existing ticket. Link it if one exists, or create a focused follow-up ticket describing the observed behavior and supporting evidence. Tell me what you found and give me the ticket URL. If you cannot file it, say why. Keep follow-up fixes separate from this implementation.

Verify each acceptance criterion and run the repository's required checks. For a regression, demonstrate the failing behavior before the fix and the passing behavior afterward. Report what changed, which checks you actually ran, their results, and anything still unverified. State assumptions and blockers plainly. Only claim outcomes you observed.

Leave commits, pushes, and pull request creation to me unless I explicitly ask you to do them.

Additional items (optional)
{{if .Vars.additional_items}}{{.Vars.additional_items}}{{else}}[Add ticket-specific instructions here]{{end}}
