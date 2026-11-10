# Patch 0017 — derived-property (F3)

## Requirement
A select property whose description contains "Derived from <Source>" (Source = another
select property's name) gets its value from the Source option's name, the part before
" · ": Type "Tech · Feature" → Domain "Tech". Applied on issue create and on every set of
a property (`PUT /api/issues/{id}/properties/{pid}`). No matching option → the derived
value is left as it is. Owner decision (2026-11-10): pick Type, Domain fills itself.

## Invariants
No migration; the rule lives in the property description, so it is visible on the
property page and declared by packages. Without such a description nothing changes.

## Touch points
`server/internal/service/vantt_derive.go`; three lines after property validation in
`service/issue.go` (create); a block after the value write in `handler/property.go`
`SetIssueProperty`; test `handler/vantt_derive_test.go`.

## Tests
See series.toml.
