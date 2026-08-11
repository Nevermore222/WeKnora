# SR_ASP Naming Optimization Design

**Date:** 2026-08-11

## Decision

Rename the existing tenant-10000 baseline agent `SR_ASP`
(`99a63205-a8f4-412f-a284-ebc043838e62`) to `ASP 综合专家` and update its
description to:

> 面向 ASP 全域知识库的综合技术专家。覆盖 JP1、ASP-BSP-RM 等 ASP 产品，提供基于知识库的检索、深度阅读、引用式问答、配置指导、操作步骤和故障排查。

The new display name follows the established `ASP-JP1 专家` and
`ASP-BSP-RM 账票专家` naming pattern while keeping `SR_ASP` as the internal
baseline identifier in documentation and configuration references.

## Scope And Invariants

- Update only `name` and `description` for the exact baseline Agent ID in
  tenant `10000`.
- Preserve the Agent ID, owner, avatar, configuration JSON, model IDs,
  knowledge-base bindings, tool list, and Web/MCP/Skills settings byte-for-byte.
- Do not modify either newly created expert Agent, any knowledge base, model,
  user, or unrelated Agent.
- Read the persisted row back after the update and verify the unchanged
  configuration hash and exact new display fields.
- Re-running the operation must update the same ID and must never create a
  second `ASP 综合专家` row.

## Failure Handling And Verification

Abort without changing anything if the baseline ID is missing, belongs to a
different tenant, is duplicated, or the pre-update configuration snapshot
cannot be captured. After the update, fail if the ID, tenant, owner, or config
hash changes unexpectedly. Confirm the two specialized Agent rows and all
knowledge-base rows retain their pre-update hashes.
