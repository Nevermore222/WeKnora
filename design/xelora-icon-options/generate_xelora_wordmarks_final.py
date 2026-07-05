from __future__ import annotations

from pathlib import Path
from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parent
OUT_DIR = ROOT / "final-wordmark-options"
NAVY = "#14253b"
NAVY_SOFT = "#223650"
GOLD = "#e2af4b"
WHITE = "#ffffff"


def font(size: int, bold: bool = False) -> ImageFont.ImageFont:
    candidates = []
    if bold:
        candidates += [
            Path("C:/Windows/Fonts/segoeuib.ttf"),
            Path("C:/Windows/Fonts/arialbd.ttf"),
        ]
    candidates += [
        Path("C:/Windows/Fonts/segoeui.ttf"),
        Path("C:/Windows/Fonts/arial.ttf"),
    ]
    for path in candidates:
        if path.exists():
            return ImageFont.truetype(str(path), size=size)
    return ImageFont.load_default()


def rgba(hex_color: str) -> tuple[int, int, int, int]:
    hex_color = hex_color.lstrip("#")
    return tuple(int(hex_color[i : i + 2], 16) for i in (0, 2, 4)) + (255,)


def make_canvas() -> Image.Image:
    return Image.new("RGBA", (2400, 720), rgba(WHITE))


def draw_wordmark_01() -> Image.Image:
    img = make_canvas()
    draw = ImageDraw.Draw(img)
    f = font(270, bold=False)
    draw.text((90, 150), "Xelora", font=f, fill=rgba(NAVY))
    draw.line((110, 520, 1440, 520), fill=rgba(NAVY), width=14)
    draw.line((110, 548, 1440, 548), fill=rgba(GOLD), width=8)
    return img


def draw_wordmark_02() -> Image.Image:
    img = make_canvas()
    draw = ImageDraw.Draw(img)
    f = font(268, bold=False)
    draw.text((92, 148), "Xelora", font=f, fill=rgba(NAVY))
    draw.line((1280, 162, 1280, 472), fill=rgba(GOLD), width=10)
    draw.line((1320, 162, 1320, 430), fill=rgba(NAVY_SOFT), width=18)
    return img


def draw_wordmark_03() -> Image.Image:
    img = make_canvas()
    draw = ImageDraw.Draw(img)
    f = font(276, bold=False)
    draw.text((94, 144), "Xelora", font=f, fill=rgba(NAVY))
    draw.line((112, 530, 980, 530), fill=rgba(NAVY), width=14)
    draw.line((1010, 530, 1470, 530), fill=rgba(GOLD), width=14)
    return img


OPTIONS = [
    ("wordmark-final-01", "Foundation Line", draw_wordmark_01),
    ("wordmark-final-02", "Vertical Accent", draw_wordmark_02),
    ("wordmark-final-03", "Split Underline", draw_wordmark_03),
]


def build_sheet() -> None:
    sheet = Image.new("RGBA", (2600, 1450), (10, 18, 29, 255))
    draw = ImageDraw.Draw(sheet)
    title = font(64, bold=True)
    sub = font(30)
    label = font(30)
    draw.text((60, 34), "Xelora final wordmark round", font=title, fill=(246, 247, 250, 255))
    draw.text((60, 112), "Clean enterprise wordmarks. No stylized X, no ambiguity, no decorative distortion.", font=sub, fill=(156, 176, 197, 255))

    y = 190
    for idx, (name, label_text, _) in enumerate(OPTIONS, start=1):
        draw.rounded_rectangle((60, y, 2520, y + 360), radius=34, fill=(24, 38, 58, 255))
        wm = Image.open(OUT_DIR / f"{name}.png").convert("RGBA").resize((1800, 540), Image.Resampling.LANCZOS)
        crop = wm.crop((0, 80, 1700, 410))
        sheet.alpha_composite(crop, (110, y + 22))
        draw.text((1880, y + 120), f"{idx}. {label_text}", font=label, fill=(246, 247, 250, 255))
        draw.text((1880, y + 168), name, font=sub, fill=(156, 176, 197, 255))
        y += 400
    sheet.save(OUT_DIR / "xelora-wordmark-sheet.png")


def write_notes() -> None:
    text = """# Xelora Final Wordmark Round

1. Foundation Line
   Most formal and stable. Best default for platform branding.

2. Vertical Accent
   Still corporate, but with a little more product identity.

3. Split Underline
   Clean and modern, slightly more designed than 01.

Recommendation:
- safest default: wordmark-final-01
- if you want a little extra identity: wordmark-final-02
"""
    (OUT_DIR / "notes.md").write_text(text, encoding="utf-8")


def main() -> None:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    for name, _label, builder in OPTIONS:
        builder().save(OUT_DIR / f"{name}.png")
    build_sheet()
    write_notes()


if __name__ == "__main__":
    main()
