# Spark Workflow for Echo CLI

This file is the execution contract for the prompt kit. The project overview and ADR define product and architecture. Milestone files define scope. GitHub issues define the unit of implementation.

## Initial Setup

1. Copy this kit to `docs/plan/`.
2. Run `/spark:onboard` once.
3. Confirm `spark doctor` is healthy.
4. Commit the planning documents before implementation begins.
5. Keep the default branch clean and green.

## Per-Milestone Flow

1. Read `project-overview.md`, the ADR, `build-sequence.md`, and the current milestone.
2. Run `/spark:plan`.
3. Create the milestone's GitHub issues using the supplied issue contracts and checkbox criteria.
4. Approve only the issues for the current milestone.
5. For each issue, run `/spark:codify`, `/spark:validate`, and `/spark:ship`.
6. Merge the issue PR before beginning dependent work.
7. Run the milestone definition of done after all milestone issues merge.
8. Do not begin the next milestone until the current milestone is green.

## Source of Truth

When documents disagree, use this order:

1. Current approved GitHub issue for implementation scope.
2. Current milestone for milestone boundaries.
3. ADR for architecture.
4. Project overview for product scope.
5. Build sequence for ordering guidance.

An issue may narrow a milestone but may not broaden it. Architectural changes require an ADR update or a new ADR before implementation.

## Branch and PR Discipline

- One issue per branch.
- One focused PR per issue.
- No drive-by refactors.
- No future milestone work.
- No generated recordings, transcripts, model weights, virtual environments, or user data.
- PR descriptions must link the issue, summarize the observable result, and list validation performed.

## When to Skip Full Ceremony

Tiny documentation corrections or typo fixes may use a direct focused branch and `/spark:ship` when no planning decision is involved. Any behavioral change, contract change, dependency change, or architecture change must have an issue and validation.

## Final Stop Condition

Milestone 7 produces Echo CLI v0.1. Stop after that release. The web app and SaaS are separate planning efforts and must not leak into this kit.
