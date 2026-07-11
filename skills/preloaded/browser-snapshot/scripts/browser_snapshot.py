#!/usr/bin/env python3
"""Browser snapshot script: navigate to a URL and capture screenshot/content.

Usage:
    python browser_snapshot.py <url> <capture_mode>

Args:
    url: Target page URL
    capture_mode: "screenshot", "content", or "both"

Outputs (written to the current working directory):
    screenshot.png   - Page screenshot (when mode is screenshot or both)
    page_content.html - Page HTML content (when mode is content or both)

Exit codes:
    0 - Success
    1 - Error (see stderr)
"""

import os
import sys
import time


def navigate_and_capture(url, capture_mode):
    """Navigate to the URL and capture output based on mode."""
    # Try Playwright first, fall back to Selenium
    try:
        return navigate_playwright(url, capture_mode)
    except ImportError:
        pass
    try:
        return navigate_selenium(url, capture_mode)
    except ImportError:
        pass
    print("Error: Neither Playwright nor Selenium is installed.", file=sys.stderr)
    print("Install with: pip install playwright && playwright install chromium", file=sys.stderr)
    print("Or: pip install selenium", file=sys.stderr)
    return 1


def navigate_playwright(url, capture_mode):
    from playwright.sync_api import sync_playwright

    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page(viewport={"width": 1280, "height": 720})
        try:
            page.goto(url, wait_until="domcontentloaded", timeout=45000)
            page.wait_for_timeout(2000)

            if capture_mode in ("screenshot", "both"):
                screenshot_path = os.path.join(os.getcwd(), "screenshot.png")
                page.screenshot(path=screenshot_path, full_page=False)
                print(f"Screenshot saved: {screenshot_path}")

            if capture_mode in ("content", "both"):
                content_path = os.path.join(os.getcwd(), "page_content.html")
                content = page.content()
                with open(content_path, "w", encoding="utf-8") as f:
                    f.write(content)
                print(f"Content saved: {content_path}")
        finally:
            browser.close()
    return 0


def navigate_selenium(url, capture_mode):
    from selenium import webdriver
    from selenium.webdriver.chrome.options import Options
    from selenium.webdriver.support.ui import WebDriverWait
    from selenium.webdriver.support import expected_conditions as EC

    options = Options()
    options.add_argument("--headless")
    options.add_argument("--no-sandbox")
    options.add_argument("--disable-dev-shm-usage")
    options.add_argument("--window-size=1280,720")

    driver = webdriver.Chrome(options=options)
    try:
        driver.set_page_load_timeout(45)
        driver.get(url)
        time.sleep(2)

        if capture_mode in ("screenshot", "both"):
            screenshot_path = os.path.join(os.getcwd(), "screenshot.png")
            driver.save_screenshot(screenshot_path)
            print(f"Screenshot saved: {screenshot_path}")

        if capture_mode in ("content", "both"):
            content_path = os.path.join(os.getcwd(), "page_content.html")
            with open(content_path, "w", encoding="utf-8") as f:
                f.write(driver.page_source)
            print(f"Content saved: {content_path}")
    finally:
        driver.quit()
    return 0


def main():
    if len(sys.argv) < 2:
        print("Usage: browser_snapshot.py <url> [capture_mode]", file=sys.stderr)
        return 1

    url = sys.argv[1]
    capture_mode = sys.argv[2] if len(sys.argv) > 2 else "screenshot"

    if capture_mode not in ("screenshot", "content", "both"):
        print(f"Invalid capture mode: {capture_mode}. Use screenshot, content, or both.", file=sys.stderr)
        return 1

    print(f"Navigating to: {url}")
    print(f"Capture mode: {capture_mode}")
    return navigate_and_capture(url, capture_mode)


if __name__ == "__main__":
    sys.exit(main())
