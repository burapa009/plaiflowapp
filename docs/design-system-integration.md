# Applying the PlaiFlow design system

`design-system.md` is the user-supplied v1 specification, adopted on 2026-09-30. It supersedes older palette, typography, spacing, and component styling guidance. The reference screenshots guide document-review composition; this specification governs its visual language.

## Implementation

- `web/app/design-tokens.css` contains the approved `--pf-*` palette and semantic/component aliases. `globals.css` imports it and maps the existing Tailwind names (`brand`, `surface`, `fg`, `muted`, `line`, `shadow-panel`) to those tokens so existing routes inherit the new foundation.
- Reuse existing native controls and shared button/form/card classes. Keep the current Next.js directory structure and outline SVG style; a design recommendation is not a requirement to install a library or reorganize the repository.
- The bundled Noto Sans Thai remains the served fallback. LINE Seed Sans TH and IBM Plex Sans Thai are preferred if available on the device; neither font has been downloaded or bundled.
- Preserve the exact palette, but use accessible semantic aliases: normal text uses `--pf-text-secondary` instead of the low-contrast decorative muted color. Primary buttons use Primary 600 with white text because Primary 500 does not meet 4.5:1 for normal-size white labels. Control boundaries use Gray 500; subtle borders remain for decorative card dividers. Primary 500 remains the brand accent. Verify contrast for each new pairing.
- Light mode is the implemented default. Dark palette tokens are reference values, not a released dark theme.

## Scope and verification

This adoption updates shared foundations and existing document styling; it is not proof that every historical page has been individually redesigned. Migrate remaining page-local styles when touching those components. The specification's search, notifications, expense creation, multi-file viewer, and other feature examples do not imply those workflows already exist. Keep visible controls tied to actual backend capabilities; do not invent credit usage, confidence, payment status, or tax values.

Run the scripts in `web/package.json` for tests, lint, typecheck and build. Check representative desktop/mobile pages, focus and disabled states, reduced motion, long Thai labels, and financial number alignment. Production deployment is a separate step.
