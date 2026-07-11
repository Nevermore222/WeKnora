# Data Model: OpenSandbox Runtime Baseline

## RuntimeModuleFamily

Represents a major capability area in the Xelora web agent runtime.

Fields:

- `id`: Stable module family identifier.
- `name`: Human-readable name such as `sandbox-execution` or `file-capability`.
- `purpose`: Short statement of user or system value.
- `stage`: Adoption stage such as `baseline`, `follow_up`, or `future`.
- `owner_scope`: Whether the module is primarily `xelora`, `provider`, or `shared_contract`.
- `status`: `planned`, `selected`, `integrating`, `replaceable`.

Relationships:

- One module family has one or more reference projects.
- One module family may depend on another module family.
- One module family can have one ownership boundary record.

Validation:

- Every in-scope module family must have exactly one primary baseline or semantic reference.
- Every baseline-stage module family must define an ownership boundary.

## ReferenceProject

Represents a concrete project or project category used as a baseline, alternative, or semantic reference.

Fields:

- `id`: Stable reference identifier.
- `name`: Project name.
- `role`: `primary_baseline`, `secondary_alternative`, `semantic_reference`, `service_reference`, or `embedded_reference`.
- `module_family_id`: The module family it supports.
- `adoption_mode`: `wrap`, `integrate`, `reference_only`, or `evaluate_later`.
- `deployment_shape`: `independent_service`, `library`, `cli`, `embedded_sdk`, or `mixed`.
- `replacement_risk`: `low`, `medium`, or `high`.
- `notes`: Constraint or caution summary.

Validation:

- A `primary_baseline` must map to an in-scope module family.
- The `sandbox-execution` module family must have one active primary baseline.
- A project cannot be both `primary_baseline` and `reference_only` for the same module family.

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
- Session workspace identity, artifact identity, and execution history must appear in at least one Xelora-owned concern set.

## AdoptionStage

Represents the staged delivery order for the runtime architecture.

Fields:

- `id`: Stable stage identifier.
- `name`: Stage name such as `baseline-runtime` or `files-and-browser`.
- `goal`: What the stage unlocks for users or maintainers.
- `included_module_families`: Ordered list of module family ids.
- `entry_criteria`: Preconditions for starting the stage.
- `exit_criteria`: Observable outcomes that mark the stage complete.

Validation:

- Each module family must appear in at least one adoption stage.
- The first stage must form a minimal useful runtime baseline.
- The first stage must include `sandbox-execution`, `execution-gateway`, `workspace-management`, and `artifact-management`.

## ContractInvariant

Represents a product-level behavior that must survive provider changes.

Fields:

- `id`: Stable invariant identifier.
- `name`: Short name such as `session-workspace-ownership`.
- `description`: Full statement of the invariant.
- `applies_to`: Related module families.
- `risk_if_broken`: Product or architectural impact.

Validation:

- Invariants tied to session workspace, job identity, artifact identity, policy ownership, and user-visible execution history must be defined before implementation work begins.
- Any provider replacement proposal must map back to the existing invariant set rather than inventing a new product contract.
