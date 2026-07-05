from __future__ import annotations

import math
from pathlib import Path
from PIL import Image, ImageDraw, ImageFilter, ImageFont


ROOT = Path(__file__).resolve().parent
BADGE_DIR = ROOT / "badge-options"
WORDMARK_DIR = ROOT / "wordmark-options"
SIZE = 1024

NAVY = "#112033"
DEEP = "#08131f"
RED = "#a5121d"
CRIMSON = "#cf2331"
WHITE = "#f3f6fa"
SILVER = "#ccd5df"
GOLD = "#e0a94b"
CYAN = "#55d7ff"


def ensure_dirs() -> None:
    for path in (BADGE_DIR, WORDMARK_DIR):
        path.mkdir(parents=True, exist_ok=True)


def rgba(hex_color: str, alpha: int = 255) -> tuple[int, int, int, int]:
    hex_color = hex_color.lstrip("#")
    return tuple(int(hex_color[i : i + 2], 16) for i in (0, 2, 4)) + (alpha,)


def load_font(size: int, bold: bool = False) -> ImageFont.ImageFont:
    choices = []
    if bold:
        choices.extend(
            [
                Path("C:/Windows/Fonts/segoeuib.ttf"),
                Path("C:/Windows/Fonts/arialbd.ttf"),
            ]
        )
    choices.extend(
        [
            Path("C:/Windows/Fonts/segoeui.ttf"),
            Path("C:/Windows/Fonts/arial.ttf"),
        ]
    )
    for path in choices:
        if path.exists():
            return ImageFont.truetype(str(path), size=size)
    return ImageFont.load_default()


def metallic_gradient(size: tuple[int, int], c1: str, c2: str, diagonal: bool = True) -> Image.Image:
    w, h = size
    base = Image.new("RGBA", size, (0, 0, 0, 0))
    pix = base.load()
    r1, g1, b1, _ = rgba(c1)
    r2, g2, b2, _ = rgba(c2)
    for y in range(h):
        for x in range(w):
            if diagonal:
                t = (x + y) / max(1, (w + h - 2))
            else:
                t = y / max(1, h - 1)
            if 0.32 < t < 0.56:
                t = min(1.0, t + 0.18)
            r = int(r1 + (r2 - r1) * t)
            g = int(g1 + (g2 - g1) * t)
            b = int(b1 + (b2 - b1) * t)
            pix[x, y] = (r, g, b, 255)
    return base


def mask_polygon(size: tuple[int, int], pts: list[tuple[float, float]]) -> Image.Image:
    mask = Image.new("L", size, 0)
    draw = ImageDraw.Draw(mask)
    draw.polygon(pts, fill=255)
    return mask


def polygon_image(size: tuple[int, int], pts: list[tuple[float, float]], c1: str, c2: str, diagonal: bool = True) -> Image.Image:
    grad = metallic_gradient(size, c1, c2, diagonal=diagonal)
    grad.putalpha(mask_polygon(size, pts))
    return grad


def radial_background() -> Image.Image:
    img = Image.new("RGBA", (SIZE, SIZE), rgba(DEEP))
    draw = ImageDraw.Draw(img)
    draw.rounded_rectangle((58, 58, 966, 966), radius=220, fill=rgba(NAVY))
    glow = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    gdraw = ImageDraw.Draw(glow)
    for i, color in enumerate((CRIMSON, CYAN, GOLD)):
        inset = 250 + i * 30
        alpha = 34 - i * 7
        gdraw.ellipse((inset, inset, SIZE - inset, SIZE - inset), outline=rgba(color, alpha), width=18)
    glow = glow.filter(ImageFilter.GaussianBlur(18))
    img.alpha_composite(glow)
    return img


def umbrella_segments(cx: int, cy: int, inner: int, outer: int, count: int, twist: float = 0.0) -> list[list[tuple[float, float]]]:
    pts = []
    for i in range(count):
        a0 = math.radians((360 / count) * i - 90 + twist)
        a1 = math.radians((360 / count) * (i + 1) - 90 + twist)
        am = (a0 + a1) / 2
        p0 = (cx + inner * math.cos(a0), cy + inner * math.sin(a0))
        p1 = (cx + outer * math.cos(am - 0.12), cy + outer * math.sin(am - 0.12))
        p2 = (cx + outer * math.cos(am + 0.12), cy + outer * math.sin(am + 0.12))
        p3 = (cx + inner * math.cos(a1), cy + inner * math.sin(a1))
        pts.append([p0, p1, p2, p3])
    return pts


def draw_x_cut(draw: ImageDraw.ImageDraw, width: int, color: tuple[int, int, int, int]) -> None:
    draw.line((290, 260, 734, 764), fill=color, width=width)
    draw.line((734, 260, 290, 764), fill=color, width=width)


def make_badge_01() -> Image.Image:
    img = radial_background()
    segs = umbrella_segments(512, 512, 92, 350, 8, twist=0)
    for i, seg in enumerate(segs):
        fill = polygon_image((SIZE, SIZE), seg, WHITE if i % 2 else CRIMSON, SILVER if i % 2 else RED, diagonal=bool(i % 2))
        img.alpha_composite(fill)
    shadow = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    sdraw = ImageDraw.Draw(shadow)
    draw_x_cut(sdraw, 138, rgba(DEEP, 255))
    shadow = shadow.filter(ImageFilter.GaussianBlur(6))
    img.alpha_composite(shadow)
    draw = ImageDraw.Draw(img)
    draw_x_cut(draw, 110, rgba(DEEP, 255))
    draw_x_cut(draw, 64, rgba(CYAN, 185))
    draw.ellipse((462, 462, 562, 562), fill=rgba(GOLD), outline=rgba(DEEP), width=14)
    return img


def make_badge_02() -> Image.Image:
    img = radial_background()
    segs = umbrella_segments(512, 512, 100, 332, 6, twist=30)
    palette = [(CRIMSON, RED), (WHITE, SILVER), (CYAN, "#176d98")]
    for i, seg in enumerate(segs):
        c1, c2 = palette[i % len(palette)]
        fill = polygon_image((SIZE, SIZE), seg, c1, c2, diagonal=(i % 2 == 0))
        img.alpha_composite(fill)
    draw = ImageDraw.Draw(img)
    draw_x_cut(draw, 92, rgba(DEEP, 255))
    draw_x_cut(draw, 50, rgba(WHITE, 220))
    draw.ellipse((482, 482, 542, 542), fill=rgba(GOLD))
    return img


def make_badge_03() -> Image.Image:
    img = radial_background()
    segs = umbrella_segments(512, 512, 128, 340, 4, twist=45)
    palette = [(WHITE, SILVER), (CRIMSON, RED), (WHITE, SILVER), (CRIMSON, RED)]
    for seg, colors in zip(segs, palette):
        fill = polygon_image((SIZE, SIZE), seg, colors[0], colors[1], diagonal=True)
        img.alpha_composite(fill)
    draw = ImageDraw.Draw(img)
    draw_x_cut(draw, 164, rgba(DEEP, 255))
    draw_x_cut(draw, 94, rgba(CYAN, 155))
    draw.rounded_rectangle((456, 456, 568, 568), radius=28, fill=rgba(NAVY), outline=rgba(WHITE), width=10)
    draw.rounded_rectangle((484, 484, 540, 540), radius=18, fill=rgba(GOLD))
    return img


def make_badge_04() -> Image.Image:
    img = radial_background()
    segs = umbrella_segments(512, 512, 80, 340, 8, twist=22.5)
    for i, seg in enumerate(segs):
        if i % 2 == 0:
            fill = polygon_image((SIZE, SIZE), seg, WHITE, SILVER, diagonal=False)
        else:
            fill = polygon_image((SIZE, SIZE), seg, CRIMSON, RED, diagonal=False)
        img.alpha_composite(fill)
    draw = ImageDraw.Draw(img)
    draw_x_cut(draw, 122, rgba(DEEP, 255))
    draw.line((290, 260, 734, 764), fill=rgba(GOLD, 180), width=18)
    draw.line((734, 260, 290, 764), fill=rgba(WHITE, 160), width=14)
    draw.ellipse((470, 470, 554, 554), fill=rgba(DEEP))
    draw.ellipse((488, 488, 536, 536), fill=rgba(CYAN))
    return img


BADGES = [
    ("badge-01", "Umbra X Core", make_badge_01),
    ("badge-02", "Hex Prism X", make_badge_02),
    ("badge-03", "Shielded X", make_badge_03),
    ("badge-04", "Radiant X Seal", make_badge_04),
]


def save_badges() -> None:
    for name, _label, builder in BADGES:
        builder().save(BADGE_DIR / f"{name}.png")


def wordmark_canvas() -> Image.Image:
    return Image.new("RGBA", (2200, 700), (255, 255, 255, 255))


def draw_swoosh(draw: ImageDraw.ImageDraw, x0: int, y0: int, color: str, width: int, lift: int) -> None:
    pts = []
    for step in range(0, 33):
        t = step / 32
        x = x0 + int(980 * t)
        y = y0 + int(math.sin(t * math.pi) * -lift) + int((t - 0.5) * 28)
        pts.append((x, y))
    draw.line(pts, fill=rgba(color), width=width, joint="curve")


def draw_initial_x(draw: ImageDraw.ImageDraw, box: tuple[int, int, int, int], c_left: str, c_right: str, accent: str) -> None:
    x1, y1, x2, y2 = box
    cx = (x1 + x2) // 2
    pts1 = [(x1 + 10, y1), (x1 + 92, y1), (cx + 30, y2), (cx - 54, y2)]
    pts2 = [(x2 - 92, y1), (x2 - 10, y1), (cx - 28, y2), (cx - 110, y2)]
    draw.polygon(pts1, fill=rgba(c_left))
    draw.polygon(pts2, fill=rgba(c_right))
    draw.polygon([(cx - 34, (y1 + y2) // 2), (cx, (y1 + y2) // 2 - 34), (cx + 34, (y1 + y2) // 2), (cx, (y1 + y2) // 2 + 34)], fill=rgba(accent))


def make_wordmark_01() -> Image.Image:
    img = wordmark_canvas()
    draw = ImageDraw.Draw(img)
    draw_initial_x(draw, (80, 118, 430, 588), NAVY, CYAN, GOLD)
    font = load_font(260)
    draw.text((380, 138), "elora", font=font, fill=rgba(NAVY))
    draw_swoosh(draw, 430, 470, NAVY, 18, 54)
    draw_swoosh(draw, 466, 496, GOLD, 12, 32)
    return img


def make_wordmark_02() -> Image.Image:
    img = wordmark_canvas()
    draw = ImageDraw.Draw(img)
    draw_initial_x(draw, (70, 110, 430, 590), RED, WHITE, NAVY)
    font = load_font(252)
    draw.text((384, 140), "elora", font=font, fill=rgba(NAVY))
    draw_swoosh(draw, 420, 490, CRIMSON, 22, 26)
    draw_swoosh(draw, 438, 516, SILVER, 10, 10)
    return img


def make_wordmark_03() -> Image.Image:
    img = wordmark_canvas()
    draw = ImageDraw.Draw(img)
    draw_initial_x(draw, (66, 102, 422, 598), NAVY, CRIMSON, GOLD)
    font = load_font(258)
    draw.text((370, 134), "elora", font=font, fill=rgba("#17283d"))
    draw.line((1080, 196, 1080, 470), fill=rgba(CYAN), width=18)
    draw.line((1116, 196, 1116, 412), fill=rgba(GOLD), width=10)
    draw_swoosh(draw, 420, 502, NAVY, 18, 48)
    return img


def make_wordmark_04() -> Image.Image:
    img = wordmark_canvas()
    draw = ImageDraw.Draw(img)
    draw_initial_x(draw, (58, 112, 420, 592), NAVY, CYAN, WHITE)
    font = load_font(250)
    draw.text((374, 142), "elora", font=font, fill=rgba(NAVY))
    draw.arc((1020, 132, 1240, 410), start=40, end=320, fill=rgba(CRIMSON), width=16)
    draw_swoosh(draw, 418, 500, NAVY, 16, 40)
    draw_swoosh(draw, 454, 522, GOLD, 10, 18)
    return img


WORDMARKS = [
    ("wordmark-01", "Classic Swoosh", make_wordmark_01),
    ("wordmark-02", "Crimson Crest", make_wordmark_02),
    ("wordmark-03", "Beacon Serifless", make_wordmark_03),
    ("wordmark-04", "Orbit Accent", make_wordmark_04),
]


def save_wordmarks() -> None:
    for name, _label, builder in WORDMARKS:
        builder().save(WORDMARK_DIR / f"{name}.png")


def build_sheet() -> None:
    sheet = Image.new("RGBA", (2600, 2100), rgba("#0a1421"))
    draw = ImageDraw.Draw(sheet)
    title = load_font(62, bold=True)
    subtitle = load_font(30)
    label = load_font(32)
    draw.text((58, 34), "Xelora badge + wordmark exploration", font=title, fill=rgba(WHITE))
    draw.text((60, 112), "Inspired by the reference badge's radial authority, rebuilt around an X-centered knowledge platform identity.", font=subtitle, fill=rgba("#9eb1c6"))

    for idx, (name, text, _builder) in enumerate(BADGES):
        x = 70 + idx * 620
        y = 200
        draw.rounded_rectangle((x, y, x + 540, y + 640), radius=46, fill=rgba(NAVY))
        icon = Image.open(BADGE_DIR / f"{name}.png").convert("RGBA").resize((420, 420), Image.Resampling.LANCZOS)
        sheet.alpha_composite(icon, (x + 60, y + 56))
        draw.text((x + 34, y + 520), f"{idx + 1}. {text}", font=label, fill=rgba(WHITE))
        draw.text((x + 34, y + 568), name, font=subtitle, fill=rgba("#98adc4"))

    for idx, (name, text, _builder) in enumerate(WORDMARKS):
        x = 70
        y = 950 + idx * 275
        draw.rounded_rectangle((x, y, x + 2460, y + 220), radius=28, fill=rgba(NAVY))
        wm = Image.open(WORDMARK_DIR / f"{name}.png").convert("RGBA").resize((1500, 477), Image.Resampling.LANCZOS)
        crop = wm.crop((0, 70, 1500, 345))
        sheet.alpha_composite(crop, (120, y - 16))
        draw.text((1780, y + 60), f"{idx + 1}. {text}", font=label, fill=rgba(WHITE))
        draw.text((1780, y + 108), name, font=subtitle, fill=rgba("#98adc4"))
    sheet.save(ROOT / "badge-wordmark-sheet.png")


def write_notes() -> None:
    notes = """# Xelora Badge and Wordmark Round 2

## Badge directions

1. Umbra X Core
   The strongest direct lift from the reference's radial authority, but translated into a cleaner X-centered enterprise badge.

2. Hex Prism X
   Slightly more geometric and modern, less aggressive than the reference.

3. Shielded X
   Feels most like a software platform seal or system crest.

4. Radiant X Seal
   Most ceremonial and emblematic; strongest badge personality.

## Wordmark directions

1. Classic Swoosh
   Best match for replacing the current Xelora wordmark structure.

2. Crimson Crest
   Matches the badge palette most directly and feels more assertive.

3. Beacon Serifless
   Strongest "platform brand" posture, cleaner and more corporate.

4. Orbit Accent
   The most decorative and motion-oriented version.

## Suggested pairings

- Safe replacement set: badge-03 + wordmark-01
- Strongest visual identity: badge-01 + wordmark-02
- Most enterprise/platform-like: badge-03 + wordmark-03
"""
    (ROOT / "badge-wordmark-notes.md").write_text(notes, encoding="utf-8")


def main() -> None:
    ensure_dirs()
    save_badges()
    save_wordmarks()
    build_sheet()
    write_notes()


if __name__ == "__main__":
    main()
