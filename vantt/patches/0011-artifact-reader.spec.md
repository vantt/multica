# Patch 0011 — artifact-reader (F8) · design D10

## Requirement
Markdown attachments (specs, copy, reports) open in `MarkdownReader`: a sticky
reading-progress bar; with ≥ 3 headings (#–###, outside code fences) an outline on
wide screens (xl) beside the text, click to jump, current section highlighted; text
column unchanged (max-w-3xl, relaxed leading). Reference: GitHub's markdown outline,
Notion/Confluence page outlines.

Not built: marking an approved version (needs version status data; attachments
already have versions).

## Touch points
`packages/views/editor/markdown-reader.tsx` (+ test), the `markdown` case of
`attachment-preview-modal.tsx` (DocumentSheet → MarkdownReader), `editor.reader.*`
strings in every locale, and the modal test's i18n mock.

## Tests
`pnpm install --frozen-lockfile --ignore-scripts && cd packages/views && npx vitest run editor locales`
