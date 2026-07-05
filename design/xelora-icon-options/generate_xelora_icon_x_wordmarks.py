from __future__ import annotations

from pathlib import Path
from PIL import Image, ImageDraw, ImageFont, ImageOps


ROOT = Path(__file__).resolve().parent
ICON_PATH = ROOT / "reference-badge-options-round2" / "ref-badge-01.png"
OUT_DIR = ROOT / "icon-x-wordmark-options"

NAVY = "#14253b"
NAVY_SOFT = "#223650"
GOLD = "#e2af4b"
WHITE = "#ffffff"
DARK_BG = "#0b1118"
CARD_BG = "#1a2838"
MUTED = "#9ab1c7"


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


def load_icon(size: int) -> Image.Image:
    return Image.open(ICON_PATH).convert("RGBA").resize((size, size), Image.Resampling.LANCZOS)


def draw_text_block(
    base: Image.Image,
    text: str,
    x: int,
    y: int,
    size: int = 270,
    color: str = NAVY,
) -> tuple[int, int]:
    draw = ImageDraw.Draw(base)
    f = font(size)
    draw.text((x, y), text, font=f, fill=rgba(color))
    bbox = draw.textbbox((x, y), text, font=f)
    return bbox[2] - bbox[0], bbox[3] - bbox[1]


def draw_wordmark_01() -> Image.Image:
    img = make_canvas()
    badge = load_icon(338)
    img.alpha_composite(badge, (88, 142))
    text_x = 400
    draw_text_block(img, "elora", text_x, 150, size=270)
    draw = ImageDraw.Draw(img)
    draw.line((102, 520, 1490, 520), fill=rgba(NAVY), width=14)
    draw.line((102, 548, 1490, 548), fill=rgba(GOLD), width=8)
    return img


def build_x_from_badge() -> Image.Image:
    icon = load_icon(820)
    x_piece = Image.new("RGBA", (820, 820), (0, 0, 0, 0))
    crop = icon.crop((150, 150, 670, 670))

    left = crop.rotate(45, resample=Image.Resampling.BICUBIC, expand=True)
    right = ImageOps.mirror(left)
    left = left.resize((330, 330), Image.Resampling.LANCZOS)
    right = right.resize((330, 330), Image.Resampling.LANCZOS)

    x_piece.alpha_composite(left, (115, 90))
    x_piece.alpha_composite(right, (375, 90))
    x_piece.alpha_composite(ImageOps.flip(left), (115, 360))
    x_piece.alpha_composite(ImageOps.flip(right), (375, 360))

    ring = ImageDraw.Draw(x_piece)
    ring.ellipse((335, 335, 485, 485), fill=rgba(GOLD), outline=rgba(NAVY_SOFT), width=20)
    return x_piece.resize((345, 345), Image.Resampling.LANCZOS)


def draw_wordmark_02() -> Image.Image:
    img = make_canvas()
    x_mark = build_x_from_badge()
    img.alpha_composite(x_mark, (82, 134))
    draw_text_block(img, "elora", 405, 150, size=270)
    draw = ImageDraw.Draw(img)
    draw.line((1230, 166, 1230, 474), fill=rgba(GOLD), width=10)
    draw.line((1268, 166, 1268, 430), fill=rgba(NAVY_SOFT), width=18)
    return img


def draw_wordmark_03() -> Image.Image:
    img = make_canvas()
    badge = load_icon(260)
    img.alpha_composite(badge, (100, 188))
    draw = ImageDraw.Draw(img)
    x_font = font(282)
    draw.text((346, 146), "X", font=x_font, fill=rgba(NAVY))
    elora_font = font(270)
    draw.text((520, 150), "elora", font=elora_font, fill=rgba(NAVY))
    draw.line((110, 530, 1010, 530), fill=rgba(NAVY), width=14)
    draw.line((1040, 530, 1490, 530), fill=rgba(GOLD), width=14)
    return img


OPTIONS = [
    ("icon-x-wordmark-01", "Badge As X", draw_wordmark_01),
    ("icon-x-wordmark-02", "Constructed X", draw_wordmark_02),
    ("icon-x-wordmark-03", "Badge + Letter X", draw_wordmark_03),
]


def build_sheet() -> None:
    sheet = Image.new("RGBA", (2600, 1450), rgba(DARK_BG))
    draw = ImageDraw.Draw(sheet)
    title = font(64, bold=True)
    sub = font(30)
    label = font(30)
    draw.text((60, 34), "Xelora icon-linked wordmark round", font=title, fill=(246, 247, 250, 255))
    draw.text((60, 112), "The X now directly references the approved red-white badge icon language.", font=sub, fill=rgba(MUTED))

    y = 190
    for idx, (name, label_text, _) in enumerate(OPTIONS, start=1):
        draw.rounded_rectangle((60, y, 2520, y + 360), radius=34, fill=rgba(CARD_BG))
        wm = Image.open(OUT_DIR / f"{name}.png").convert("RGBA").resize((1800, 540), Image.Resampling.LANCZOS)
        crop = wm.crop((0, 80, 1700, 410))
        sheet.alpha_composite(crop, (110, y + 22))
        draw.text((1880, y + 120), f"{idx}. {label_text}", font=label, fill=(246, 247, 250, 255))
        draw.text((1880, y + 168), name, font=sub, fill=rgba(MUTED))
        y += 400

    sheet.save(OUT_DIR / "xelora-icon-x-wordmark-sheet.png")


def write_notes() -> None:
    text = """# Xelora Icon-Linked Wordmark Round

1. Badge As X
   Directly treats the approved badge icon as the first character.

2. Constructed X
   Rebuilds an explicit X from the icon's segmented badge language.

3. Badge + Letter X
   Keeps a normal X, but pairs it with a smaller badge accent to preserve continuity.

Recommendation:
- strongest icon-to-wordmark consistency: icon-x-wordmark-01
- most typographic balance: icon-x-wordmark-02
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
