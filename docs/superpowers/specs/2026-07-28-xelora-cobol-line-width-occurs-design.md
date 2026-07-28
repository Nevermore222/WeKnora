# Xelora COBOL Line Width and OCCURS Design

## Goal

Correct the Xelora SR6 agent output without adding semantic rewriting to n8n. Generated `.COBOL` and `.CBL` source must compile as fixed-format COBOL, stay within 60 display columns, separate level numbers from data names, and model `OCCURS` from each verification case's actual data structure.

## Confirmed Failures

- The latest `BT232B01` response contains 27 physical lines wider than 60 display columns. Its longest Japanese comment is 123 display columns.
- The shared working-storage block contains `03RESULT4`, so the level number and data name are not separated. The same defect appears in more than one batch.
- `BT232B18` contains `01 CYOZAI4 PIC S9(10) OCCURS 10` and other `01` elementary items followed by an `OCCURS` continuation. These are invalid representations of the required table hierarchy.
- The imported SR6 prompt still describes a 50-column comment limit and a 72-column code limit. Its existing `OCCURS` guidance is long and lower priority, so the model does not apply it reliably.

## Agent Contract

### Physical Line Width

- Apply the new 60-display-column limit only to `.COBOL` and `.CBL` output.
- Every physical line, including comments and non-comment code, must be at most 60 display columns.
- Count ASCII characters, digits, and spaces as one display column. Count Japanese and other full-width characters as two display columns.
- This rule overrides the imported prompt's 50-column comment and 72-column COBOL code limits. Existing shorter comment borders remain valid.
- `.CL` and `.PF` retain their existing line-width rules.

### Fixed-Format Levels

- Keep the existing column-7 comment indicator rule.
- Every data-description level number must be followed by at least one ASCII space before the data name or special entry.
- Forms such as `03RESULT4` and `01NAME` are forbidden.
- Continuation lines must preserve fixed-format COBOL indentation while also remaining within 60 display columns.

### OCCURS Structure

- A level-01 entry is the root group and must not carry `OCCURS`.
- Do not force every table into one synthetic hierarchy such as `01 -> 05 -> 10`.
- Derive the hierarchy, names, occurrence counts, PIC clauses, signs, decimal positions, and usage from the current verification case's small classification, parameter configuration, input data, and expected result.
- Use subordinate levels such as `02`, `03`, and `05` as required by the actual structure. Level gaps are allowed when valid for the source structure.
- `OCCURS` may be attached to a subordinate group item, for example `02 WK-GOKEI-TBL OCCURS 4`.
- `OCCURS` may also be attached to a subordinate elementary item with `PIC` when the actual case requires an elementary array, for example `05 WK-GOKEI-T3 PIC 9(04) OCCURS 29`.
- Nested `OCCURS` is allowed when the source structure requires multiple dimensions.
- Procedure Division references must use the actual defined element name and the correct number of subscripts. Do not invent convenience aliases or reference a group when an elementary value is required.
- Apply `PACKED-DECIMAL`, `COMP`, signed PIC, and decimal PIC only when required by the current case. The table hierarchy must not erase or generalize the required element type.

## Workflow Boundary

- n8n continues to extract, aggregate, persist, and normalize generated files.
- n8n must not rewrite line width, level numbers, PIC clauses, `OCCURS`, subscripts, or COBOL semantics.
- The original Dify workflow and source SR6 YAML remain unchanged.
- The Xelora agent upsert script imports the source prompt and appends the higher-priority contract above.

## Verification

- Extend the agent configuration test to require the new contract markers.
- Extend the live smoke test with table and scalar cases that require subordinate and nested table definitions.
- Parse the returned COBOL block in the smoke test and fail when:
  - any physical COBOL line exceeds 60 display columns;
  - any data-description level is immediately followed by a data-name character;
  - any level-01 entry contains `OCCURS`;
  - a required table lacks a subordinate `OCCURS` entry and typed elementary item;
  - indexed references do not match the declared element name or dimensions.
- These checks are test-time regression checks only. They are not added to the production n8n workflow.
- Upsert the agent and run a live API smoke test before regenerating the workflow JSON.

## Non-Goals

- Do not apply the 60-column rule to `.CL` or `.PF`.
- Do not add a second validation agent.
- Do not add production post-processing that attempts to repair generated COBOL.
- Do not impose one universal OCCURS template across unrelated verification cases.
