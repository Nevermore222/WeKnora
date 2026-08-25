# Knavis Main Logo Integration Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add the approved Knavis light/dark main logo as an internal build profile while keeping the default public build on Xelora.

**Architecture:** Keep one frontend implementation and select the main logo at build time with VITE_BRAND_PROFILE. The public profile uses the existing Xelora assets; the internal profile uses Knavis assets. Only the login/sidebar main logo changes in this slice; favicon, desktop appicon, provider icons, technical identifiers, and agent/RAG behavior remain unchanged.

**Tech Stack:** Vue 3, TypeScript, Vite, PowerShell, Bash, PNG/SVG assets.

---

### Task 1: Add the approved Knavis assets

**Files:**
- Create: frontend/src/assets/img/knavis.png
- Create: frontend/src/assets/img/knavis-dark.png
- Keep source: design/knavis-icon-options/knavis-light.svg
- Keep source: design/knavis-icon-options/knavis-dark.svg

- [ ] Copy the approved transparent 1303x504 PNG candidates into the frontend asset directory.
- [ ] Verify both copies are 1303x504 and retain transparent corners.

### Task 2: Add the build-time brand profile

**Files:**
- Create: frontend/src/config/brandProfile.ts

- [ ] Define public and internal profile values with display name and light/dark main-logo imports.
- [ ] Default invalid or missing VITE_BRAND_PROFILE values to public.
- [ ] Keep the profile limited to main-logo selection so unrelated icons and technical keys cannot drift.

### Task 3: Wire the login and sidebar main logos

**Files:**
- Modify: frontend/src/views/auth/Login.vue
- Modify: frontend/src/components/menu.vue

- [ ] Replace direct Xelora logo imports with the shared brand profile.
- [ ] Use the selected display name for the login logo alt text.
- [ ] Preserve existing light/dark theme selection and all other platform logo mappings.

### Task 4: Expose profile selection through frontend builds

**Files:**
- Modify: scripts/build-personal.ps1
- Modify: scripts/build_frontend_dist.sh

- [ ] Add a public/internal profile parameter to the Windows Personal frontend build path.
- [ ] Pass the same profile through the Docker/static frontend build path using XELORA_BRAND_PROFILE.
- [ ] Preserve public as the default so existing Xelora builds remain unchanged.

### Task 5: Verify both profiles

**Files:**
- Test: frontend build and type-check output

- [ ] Run the frontend type checker.
- [ ] Build once with VITE_BRAND_PROFILE=public and once with VITE_BRAND_PROFILE=internal.
- [ ] Confirm public output contains Xelora logo assets and internal output contains Knavis logo assets.
- [ ] Confirm no favicon, desktop appicon, provider icon, prompt, RAG, or tool files changed.
