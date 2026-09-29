# PlaiFlow project conventions

For UI work (pages, components, styling, responsive behavior, or visual reviews), read `docs/design-system.md` and `docs/design-system-integration.md` before editing. The user-adopted PlaiFlow Design System v1 replaces older visual guidance; apply it to new work and components you touch.

Use `web/app/design-tokens.css` as the runtime palette and the existing shared classes in `web/app/globals.css`. Preserve backend authorization, review acknowledgements, feature gates, and uncertain OCR values when changing presentation. A screenshot or design specification does not authorize adding billing, tax, or expense-posting behavior.

Use `git --git-dir=.git.codex --work-tree=.` in this repository. Preserve unrelated worktree changes and distinguish local verification from deployment and end-to-end evidence.
