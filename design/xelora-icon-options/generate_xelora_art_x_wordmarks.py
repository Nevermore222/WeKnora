from __future__ import annotations

from pathlib import Path
from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parent
OUT_DIR = ROOT / "art-x-wordmark-options"

WHITE = "#ffffff"
NAVY = "#14253b"
NAVY_SOFT = "#223650"
NAVY_LIGHT = "#2f4866"
GOLD = "#e2af4b"
RED = "#b31724"
RED_DARK = "#6f0b14"
SILVER = "#d9dde3"
SILVER_DARK = "#9ca8b7"
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


def rgba(hex_color: str, alpha: int = 255) -> tuple[int, int, int, int]:
    hex_color = hex_color.lstrip("#")
    return tuple(int(hex_color[i : i + 2], 16) for i in (0, 2, 4)) + (alpha,)


def make_canvas() -> Image.Image:
    return Image.new("RGBA", (2400, 760), rgba(WHITE))


def poly(draw: ImageDraw.ImageDraw, points: list[tuple[int, int]], fill: str, outline: str | None = None, width: int = 0) -> None:
    draw.polygon(points, fill=rgba(fill))
    if outline:
        draw.line(points + [points[0]], fill=rgba(outline), width=width)


def quad_point(
    p0: tuple[float, float],
    p1: tuple[float, float],
    p2: tuple[float, float],
    t: float,
) -> tuple[float, float]:
    mt = 1 - t
    x = mt * mt * p0[0] + 2 * mt * t * p1[0] + t * t * p2[0]
    y = mt * mt * p0[1] + 2 * mt * t * p1[1] + t * t * p2[1]
    return x, y


def bezier_strip(
    draw: ImageDraw.ImageDraw,
    top_curve: list[tuple[float, float]],
    bottom_curve: list[tuple[float, float]],
    fill: str,
) -> None:
    polygon = [(int(x), int(y)) for x, y in top_curve]
    polygon.extend((int(x), int(y)) for x, y in reversed(bottom_curve))
    draw.polygon(polygon, fill=rgba(fill))


def sample_quad(
    p0: tuple[float, float],
    p1: tuple[float, float],
    p2: tuple[float, float],
    steps: int = 60,
) -> list[tuple[float, float]]:
    return [quad_point(p0, p1, p2, i / steps) for i in range(steps + 1)]


def draw_wave_lines(draw: ImageDraw.ImageDraw) -> None:
    navy_top = sample_quad((334, 590), (888, 558), (1368, 580), 120)
    navy_bottom = sample_quad((336, 595), (950, 606), (1358, 586), 120)
    gold_top = sample_quad((706, 578), (1036, 570), (1326, 580), 100)
    gold_bottom = sample_quad((708, 581), (1042, 585), (1320, 583), 100)

    bezier_strip(draw, navy_top, navy_bottom, NAVY)
    bezier_strip(draw, gold_top, gold_bottom, GOLD)


def draw_shard_x(draw: ImageDraw.ImageDraw, x: int, y: int, scale: float, style: str) -> tuple[int, int]:
    w = int(360 * scale)
    h = int(430 * scale)
    cx = x + w // 2
    cy = y + h // 2

    if style == "metal":
        left_outer = [(x + 20, y + 25), (x + 120, y + 25), (cx + 10, cy), (x + 130, y + h - 25), (x + 30, y + h - 25), (cx - 85, cy)]
        left_inner_1 = [(x + 36, y + 42), (x + 106, y + 42), (cx - 18, cy - 18), (cx - 70, cy)]
        left_inner_2 = [(cx - 70, cy), (cx - 18, cy - 18), (x + 112, y + h - 42), (x + 44, y + h - 42)]
        right_outer = [(x + w - 20, y + 25), (x + w - 120, y + 25), (cx - 10, cy), (x + w - 130, y + h - 25), (x + w - 30, y + h - 25), (cx + 85, cy)]
        right_inner_1 = [(x + w - 36, y + 42), (x + w - 106, y + 42), (cx + 18, cy - 18), (cx + 70, cy)]
        right_inner_2 = [(cx + 70, cy), (cx + 18, cy - 18), (x + w - 112, y + h - 42), (x + w - 44, y + h - 42)]
        poly(draw, left_outer, RED_DARK, NAVY_SOFT, 5)
        poly(draw, left_inner_1, SILVER)
        poly(draw, left_inner_2, RED)
        poly(draw, right_outer, NAVY_SOFT, NAVY_SOFT, 5)
        poly(draw, right_inner_1, RED)
        poly(draw, right_inner_2, SILVER)
        draw.ellipse((cx - 24, cy - 24, cx + 24, cy + 24), fill=rgba(GOLD), outline=rgba(NAVY_SOFT), width=5)

    if style == "wide":
        left = [(x + 18, y + 18), (x + 118, y + 18), (cx + 14, cy), (x + 136, y + h - 18), (x + 32, y + h - 18), (cx - 82, cy)]
        right = [(x + w - 18, y + 18), (x + w - 118, y + 18), (cx - 14, cy), (x + w - 136, y + h - 18), (x + w - 32, y + h - 18), (cx + 82, cy)]
        poly(draw, left, RED, NAVY_SOFT, 6)
        poly(draw, right, SILVER, NAVY_SOFT, 6)
        poly(draw, [(x + 54, y + 36), (x + 110, y + 36), (cx - 6, cy - 8), (cx - 62, cy)], "#ff6472")
        poly(draw, [(x + w - 54, y + 36), (x + w - 110, y + 36), (cx + 6, cy - 8), (cx + 62, cy)], "#ffffff")
        draw.line((x + 60, y + h - 44, cx - 18, cy + 10), fill=rgba(SILVER), width=4)
        draw.line((x + w - 60, y + h - 44, cx + 18, cy + 10), fill=rgba("#f2f4f7"), width=4)

    if style == "slash":
        left = [(x + 46, y + 10), (x + 126, y + 10), (cx + 20, cy - 12), (x + 184, y + h - 10), (x + 102, y + h - 10), (cx - 18, cy + 12)]
        right = [(x + w - 46, y + 10), (x + w - 126, y + 10), (cx - 20, cy - 12), (x + w - 184, y + h - 10), (x + w - 102, y + h - 10), (cx + 18, cy + 12)]
        poly(draw, left, NAVY, NAVY_SOFT, 6)
        poly(draw, right, RED, NAVY_SOFT, 6)
        poly(draw, [(x + 62, y + 28), (x + 114, y + 28), (cx - 8, cy - 28), (cx - 40, cy - 10)], NAVY_LIGHT)
        poly(draw, [(x + w - 62, y + 28), (x + w - 114, y + 28), (cx + 8, cy - 28), (cx + 40, cy - 10)], "#ff5d6f")
        draw.line((cx - 8, cy - 8, x + 120, y + h - 46), fill=rgba(GOLD), width=4)

    if style == "formal":
        left = [(x + 34, y + 24), (x + 118, y + 24), (cx + 10, cy), (x + 126, y + h - 24), (x + 42, y + h - 24), (cx - 66, cy)]
        right = [(x + w - 34, y + 24), (x + w - 118, y + 24), (cx - 10, cy), (x + w - 126, y + h - 24), (x + w - 42, y + h - 24), (cx + 66, cy)]
        poly(draw, left, SILVER, NAVY_SOFT, 5)
        poly(draw, right, RED_DARK, NAVY_SOFT, 5)
        poly(draw, [(x + 48, y + 40), (x + 102, y + 40), (cx - 18, cy - 18), (cx - 58, cy)], "#ffffff")
        poly(draw, [(x + w - 48, y + 40), (x + w - 102, y + 40), (cx + 18, cy - 18), (cx + 58, cy)], RED)
        draw.line((x + 84, y + h - 44, cx - 4, cy + 4), fill=rgba(SILVER_DARK), width=3)
        draw.line((x + w - 84, y + h - 44, cx + 4, cy + 4), fill=rgba("#ff7b87"), width=3)

    return w, h


def draw_wordmark(style_name: str, x_scale: float, text_y: int, text_size: int, line_mode: str, text: str = "elora") -> Image.Image:
    img = make_canvas()
    draw = ImageDraw.Draw(img)
    x_w, x_h = draw_shard_x(draw, 96, 120, x_scale, style_name)
    f = font(text_size)
    text_x = 96 + x_w - 40
    draw.text((text_x, text_y), text, font=f, fill=rgba(NAVY))

    if line_mode == "full":
        draw.line((108, 560, 1530, 560), fill=rgba(NAVY), width=14)
        draw.line((108, 588, 1530, 588), fill=rgba(GOLD), width=8)
    if line_mode == "wave":
        draw_wave_lines(draw)
    if line_mode == "split":
        draw.line((108, 560, 920, 560), fill=rgba(NAVY), width=14)
        draw.line((948, 560, 1530, 560), fill=rgba(GOLD), width=14)
    if line_mode == "none":
        pass

    return img


OPTIONS = [
    ("art-x-wordmark-01", "Monument Metal X", lambda: draw_wordmark("metal", 1.02, 188, 248, "wave", text="ELORA")),
    ("art-x-wordmark-02", "Wide Faceted X", lambda: draw_wordmark("wide", 1.28, 186, 244, "split")),
    ("art-x-wordmark-03", "Blade Slash X", lambda: draw_wordmark("slash", 1.24, 184, 248, "none")),
    ("art-x-wordmark-04", "Formal Crest X", lambda: draw_wordmark("formal", 1.20, 184, 248, "full")),
]


def build_sheet() -> None:
    sheet = Image.new("RGBA", (2660, 1900), rgba(DARK_BG))
    draw = ImageDraw.Draw(sheet)
    title = font(64, bold=True)
    sub = font(30)
    label = font(28)
    draw.text((60, 34), "Xelora art-X wordmark round", font=title, fill=(246, 247, 250, 255))
    draw.text((60, 112), "Oversized artistic X. The X leads, elora supports.", font=sub, fill=rgba(MUTED))

    y = 180
    for idx, (name, label_text, _) in enumerate(OPTIONS, start=1):
        draw.rounded_rectangle((60, y, 2580, y + 390), radius=34, fill=rgba(CARD_BG))
        wm = Image.open(OUT_DIR / f"{name}.png").convert("RGBA").resize((1900, 600), Image.Resampling.LANCZOS)
        crop = wm.crop((0, 90, 1800, 440))
        sheet.alpha_composite(crop, (95, y + 20))
        draw.text((1920, y + 122), f"{idx}. {label_text}", font=label, fill=(246, 247, 250, 255))
        draw.text((1920, y + 166), name, font=sub, fill=rgba(MUTED))
        y += 420

    sheet.save(OUT_DIR / "xelora-art-x-wordmark-sheet.png")


def write_notes() -> None:
    text = """# Xelora Art-X Wordmark Round

1. Monument Metal X
   Heaviest and most emblematic. Strong first-letter presence.

2. Wide Faceted X
   Broader silhouette, cleaner read at brand-header scale.

3. Blade Slash X
   Sharper and more aggressive, least corporate.

4. Formal Crest X
   Most restrained version while still making X the hero.

Recommendation:
- first try: art-x-wordmark-01
- if 01 feels too heavy: art-x-wordmark-02
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
