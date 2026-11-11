# Phase 05 — F3 workflow-by-type (K7) · design D6

## Reference
Jira workflow scheme (issue type → workflow). Asked for upstream in multica-ai/multica#5524.

## Requirement
Per project: pick one select property (default Type) and rules option → workflow.
Resolution: matching rule → `project.workflow_id` → workspace Default. Changing the
property or the project moves the issue to the first column of the new workflow (as moving
projects does today); an agent cannot change Type while the issue is in a gate.
API `GET/PUT /api/projects/{id}/workflow-rules`. UI settings table "When Type is → Use
workflow → first column handler", "Anything else → default".

## Steps
- [ ] Spec `0003-workflow-by-type.spec.md`; table `project_property_workflow`.
- [ ] Change `issueworkflow.ForProject`/`ForProjectRow` and their ~10 callers to take the issue.
- [ ] Tests for resolution, Type change, project move, gate interaction.
- [ ] fgOPC `opc.mjs`: `.fgopc.yaml` `projects.<p>.workflow_rules`; skip with a warning on an
      upstream image.

## Done when
One momentry project runs delivery for Tech types and content for Marketing types, columns
hand off to the right agents.

## Rollback
Remove the patch from `series.toml` (or set the owner instance back to the previous
`MULTICA_IMAGE_TAG`). Migrations only add, so the previous image keeps working on the data.
