import re
import ssl
import urllib.request
from pathlib import Path

OUT = Path(r"E:\Xelora\WeKnora\docs\login-references")
OUT.mkdir(parents=True, exist_ok=True)
CTX = ssl.create_default_context()
HEADERS = {"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"}

PAGES = {
    "01-aceternity-gradient": "https://ui.aceternity.com/blocks/login-and-signup-sections/login-form-with-gradient",
    "02-21st-login-card": "https://21st.dev/@ruixen.ui/components/login-card",
    "03-magicui-home": "https://magicui.design/",
}

for name, url in PAGES.items():
    try:
        html = urllib.request.urlopen(
            urllib.request.Request(url, headers=HEADERS), context=CTX, timeout=30
        ).read().decode("utf-8", "ignore")
        imgs = re.findall(
            r'<meta[^>]+property=["\']og:image["\'][^>]+content=["\']([^"\']+)["\']', html
        )
        if not imgs:
            imgs = re.findall(
                r'<meta[^>]+content=["\']([^"\']+)["\'][^>]+property=["\']og:image["\']', html
            )
        print(name, "og:", imgs[:1])
        if imgs:
            data = urllib.request.urlopen(
                urllib.request.Request(imgs[0], headers=HEADERS), context=CTX, timeout=30
            ).read()
            ext = ".png" if ".png" in imgs[0].lower() else ".jpg"
            path = OUT / f"{name}{ext}"
            path.write_bytes(data)
            print(" saved", path.name, len(data))
    except Exception as exc:
        print(name, "ERR", exc)
