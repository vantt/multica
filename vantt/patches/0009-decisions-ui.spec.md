# Patch 0009 — decisions-ui (F5, UI) · design D4

Needs 0008. `DecisionPanel` above `ApprovalPanel` on the issue page: question, due date,
options as a radio group (recommended marked and preselected), optional note, "Answer:
<option>". `useAnswerDecision` records the answer then posts
`**<question>** → <option>` (+ note) as a comment. Strings `issues.workflows.decision.*`
in every locale. Core module `@multica/core/decisions` (plain query options object).

## Tests
`pnpm install --frozen-lockfile --ignore-scripts && cd packages/views && npx vitest run workflows locales`
