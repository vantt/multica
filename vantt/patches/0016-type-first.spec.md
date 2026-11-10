# Patch 0016 — type-first (F3)

Needs 0005/0006 (workflow rules).

## Requirement
In a project whose workflow rules read a property (Type), that property is always shown:
in the create-issue form (manual mode) as a pill next to the other fields, and on the
issue page as a property row, without going through "Add property" / "… → Custom
properties". Other custom properties keep upstream's progressive disclosure.
Reference: Jira's create dialog puts the work type first. Owner (2026-11-10): "không
thấy property type và domain trong issues".

## Touch points
`ruleProperties` in `packages/core/workflow-rules/index.ts` (+ test); in
`packages/views/modals/create-issue.tsx` and `issues/components/issue-detail.tsx` one
rules query and one extra condition in the custom-property row filter.

## Tests
See series.toml.
