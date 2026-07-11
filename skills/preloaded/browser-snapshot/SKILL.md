# Browser Snapshot Skill

This skill provides browser automation capabilities for navigating to URLs
and capturing screenshots or page content as artifacts.

## Usage

The `browser_navigate` agent tool invokes this skill's script to:

1. Open a headless browser
2. Navigate to the target URL
3. Capture a screenshot (PNG) and/or page content (HTML/Markdown)
4. Write output files to the working directory

## Script

- **Path**: `scripts/browser_snapshot.py`
- **Arguments**: `<url> <capture_mode>`
  - `url`: Target page URL
  - `capture_mode`: `screenshot`, `content`, or `both`
- **Output**: Screenshot file (`screenshot.png`) and/or content file (`page_content.html`)

## Requirements

The sandbox Docker image must contain a headless browser binary (Chromium)
and a compatible automation library (Playwright or Selenium).
