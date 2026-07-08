# Data Model: Runtime Reference Architecture

## RuntimeModuleFamily

Represents a major capability area in the Xelora web agent runtime.

Fields:

- `id`: Stable module family identifier.
- `name`: Human-readable name such as `sandbox-execution` or `browser-automation`.
- `purpose`: Short statement of user or system value.
- `stage`: Adoption stage such as `baseline`, `follow-up`, or `future`.
- `owner_scope`: Whether the module is primarily `xelora`, `provider`, or `shared-contract`.
- `status`: `planned`, `selected`, `integrated`, `replaceable`.

Relationships:

- One module family has one or more reference projects.
- One module family may depend on another module family.

Validation:

- Every in-scope module family must have exactly one primary baseline.
- `owner_scope` must not be empty.

## ReferenceProject

Represents a concrete project or project category used as a baseline, alternative, or semantic reference.

Fields:

- `id`: Stable reference identifier.
- `name`: Project name.
- `role`: `primary-baseline`, `secondary-alternative`, `semantic-reference`, `service-reference`, `embedded-reference`.
- `module_family_id`: The module family it supports.
- `adoption_mode`: `wrap`, `integrate`, `reference-only`, `evaluate-later`.
- `deployment_shape`: `independent-service`, `library`, `cli`, `embedded-sdk`, `mixed`.
- `replacement_risk`: `low`, `medium`, `high`.
- `notes`: Constraint or caution summary.

Validation:

- A `primary-baseline` must map to an in-scope module family.
- A project cannot be both `primary-baseline` and `reference-only` for the same module family.

## OwnershipBoundary

Represents the contract split between Xelora and an external provider.

Fields:

- `id`: Stable boundary identifier.
- `module_family_id`: Related module family.
- `xelora_owned_concerns`: List of product concerns kept in Xelora.
- `provider_owned_concerns`: List of delegated concerns.
- `must_remain_stable`: List of invariants that cannot change when a provider is replaced.

Validation:

- Every primary module family must have one ownership boundary record.
- `must_remain_stable` cannot be empty for replaceable providers.

## AdoptionStage

Represents the staged delivery order for the runtime architecture.

Fields:

- `id`: Stable stage identifier.
- `name`: Stage name such as `baseline-runtime` or `browser-and-files`.
- `goal`: What the stage unlocks for users or maintainers.
- `included_module_families`: Ordered list of module family ids.
- `entry_criteria`: Preconditions for starting the stage.
- `exit_criteria`: Observable outcomes that mark the stage complete.

Validation:

- Each module family must appear in at least one adoption stage.
- The first stage must form a minimal useful runtime baseline.

## ContractInvariant

Represents a product-level behavior that must survive provider changes.

Fields:

- `id`: Stable invariant identifier.
- `name`: Short name such as `session-workspace-ownership`.
- `description`: Full statement of the invariant.
- `applies_to`: Related module families.
- `risk_if_broken`: Product or architectural impact.

Validation:

- Invariants tied to session workspace, job identity, artifact identity, and user-visible history must be covered before implementation planning continues.
