from __future__ import annotations

import math
from pathlib import Path
from PIL import Image, ImageDraw, ImageFilter, ImageFont


ROOT = Path(__file__).resolve().parent
OUT_DIR = ROOT / "reference-badge-options-round2"
SIZE = 1024

BLACK = "#000000"
SILVER_1 = "#f8f9fb"
SILVER_2 = "#d6dde5"
SILVER_3 = "#aab3bd"
RED_1 = "#d01823"
RED_2 = "#9b0d17"
RED_3 = "#5e0810"
GLOW = "#f0b849"
OUTLINE = "#d8dde3"


def rgba(value: str, alpha: int = 255) -> tuple[int, int, int, int]:
    value = value.lstrip("#")
    return tuple(int(value[i : i + 2], 16) for i in (0, 2, 4)) + (alpha,)


def ensure_dirs() -> None:
    OUT_DIR.mkdir(parents=True, exist_ok=True)


def font(size: int, bold: bool = False) -> ImageFont.ImageFont:
    picks = []
    if bold:
        picks.extend(
            [
                Path("C:/Windows/Fonts/segoeuib.ttf"),
                Path("C:/Windows/Fonts/arialbd.ttf"),
            ]
        )
    picks.extend(
        [
            Path("C:/Windows/Fonts/segoeui.ttf"),
            Path("C:/Windows/Fonts/arial.ttf"),
        ]
    )
    for path in picks:
        if path.exists():
            return ImageFont.truetype(str(path), size=size)
    return ImageFont.load_default()


def poly_mask(points: list[tuple[float, float]]) -> Image.Image:
    mask = Image.new("L", (SIZE, SIZE), 0)
    draw = ImageDraw.Draw(mask)
    draw.polygon(points, fill=255)
    return mask


def gradient(c1: str, c2: str, c3: str | None = None, diagonal: bool = True) -> Image.Image:
    img = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    px = img.load()
    a = rgba(c1)
    b = rgba(c2)
    c = rgba(c3) if c3 else None
    for y in range(SIZE):
        for x in range(SIZE):
            t = (x + y) / (SIZE * 2 - 2) if diagonal else y / (SIZE - 1)
            if c is None:
                r = int(a[0] + (b[0] - a[0]) * t)
                g = int(a[1] + (b[1] - a[1]) * t)
                bl = int(a[2] + (b[2] - a[2]) * t)
            else:
                if t < 0.55:
                    tt = t / 0.55
                    r = int(a[0] + (b[0] - a[0]) * tt)
                    g = int(a[1] + (b[1] - a[1]) * tt)
                    bl = int(a[2] + (b[2] - a[2]) * tt)
                else:
                    tt = (t - 0.55) / 0.45
                    r = int(b[0] + (c[0] - b[0]) * tt)
                    g = int(b[1] + (c[1] - b[1]) * tt)
                    bl = int(b[2] + (c[2] - b[2]) * tt)
            px[x, y] = (r, g, bl, 255)
    return img


def brushed_overlay(alpha: int, invert: bool = False) -> Image.Image:
    img = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    color = rgba("#000000" if invert else "#ffffff", alpha)
    for i in range(-SIZE, SIZE * 2, 26):
        draw.line((i, 0, i - SIZE // 2, SIZE), fill=color, width=2)
    for i in range(-80, SIZE + 80, 64):
        draw.arc((i - 120, i - 40, i + 220, i + 110), start=20, end=145, fill=color, width=2)
    return img.filter(ImageFilter.GaussianBlur(0.8))


def make_blade_points(
    cx: float,
    cy: float,
    inner_r: float,
    outer_r: float,
    angle_deg: float,
    width_deg: float,
    notch: float,
    waist: float,
    tip_pull: float,
) -> list[tuple[float, float]]:
    a = math.radians(angle_deg)
    spread = math.radians(width_deg / 2)
    left = a - spread
    right = a + spread
    mid_left = a - spread * waist
    mid_right = a + spread * waist

    p0 = (cx + inner_r * math.cos(left), cy + inner_r * math.sin(left))
    p1 = (cx + (outer_r - tip_pull) * math.cos(mid_left), cy + (outer_r - tip_pull) * math.sin(mid_left))
    p2 = (cx + outer_r * math.cos(a), cy + outer_r * math.sin(a))
    p3 = (cx + (outer_r - tip_pull) * math.cos(mid_right), cy + (outer_r - tip_pull) * math.sin(mid_right))
    p4 = (cx + inner_r * math.cos(right), cy + inner_r * math.sin(right))
    return [p0, p1, p2, p3, p4]


def blade_shadow(points: list[tuple[float, float]], offset: tuple[int, int]) -> Image.Image:
    img = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    shifted = [(x + offset[0], y + offset[1]) for x, y in points]
    draw.polygon(shifted, fill=rgba("#000000", 165))
    return img.filter(ImageFilter.GaussianBlur(18))


def render_blade(
    base: Image.Image,
    points: list[tuple[float, float]],
    red: bool,
    diagonal: bool,
    shadow_offset: tuple[int, int],
) -> None:
    base.alpha_composite(blade_shadow(points, shadow_offset))

    # richer metallic raster fill
    if red:
        fill = gradient("#f22b39", RED_1, RED_3, diagonal=diagonal)
    else:
        fill = gradient("#ffffff", SILVER_1, SILVER_3, diagonal=diagonal)
    mask = poly_mask(points)
    fill.putalpha(mask)
    base.alpha_composite(fill)

    draw = ImageDraw.Draw(base)
    inner = points[0]
    left_mid = points[1]
    tip = points[2]
    right_mid = points[3]
    edge = points[4]

    if red:
        draw.polygon([inner, left_mid, tip], fill=rgba("#ff4a57", 78))
        draw.polygon([inner, right_mid, tip], fill=rgba("#4b060d", 92))
        draw.line((inner[0], inner[1], tip[0], tip[1]), fill=rgba("#ffd3d7", 75), width=4)
        draw.arc(
            (tip[0] - 120, tip[1] - 110, tip[0] + 60, tip[1] + 60),
            start=200,
            end=302,
            fill=rgba("#ffffff", 35),
            width=3,
        )
    else:
        draw.polygon([inner, left_mid, tip], fill=rgba("#ffffff", 88))
        draw.polygon([inner, right_mid, tip], fill=rgba("#96a1ad", 82))
        draw.line((inner[0], inner[1], tip[0], tip[1]), fill=rgba("#ffffff", 135), width=5)
        draw.line((left_mid[0], left_mid[1], edge[0], edge[1]), fill=rgba("#ffffff", 70), width=3)
        draw.arc(
            (tip[0] - 140, tip[1] - 120, tip[0] + 80, tip[1] + 80),
            start=196,
            end=304,
            fill=rgba("#ffffff", 45),
            width=4,
        )

    loop = points + [points[0]]
    draw.line(loop, fill=rgba("#111316", 255), width=12, joint="curve")
    draw.line(loop, fill=rgba(OUTLINE, 140), width=3, joint="curve")


def center_hub(base: Image.Image, variant: int) -> None:
    glow = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    gdraw = ImageDraw.Draw(glow)
    gdraw.ellipse((430, 430, 594, 594), fill=rgba(GLOW, 72))
    glow = glow.filter(ImageFilter.GaussianBlur(32))
    base.alpha_composite(glow)

    draw = ImageDraw.Draw(base)
    if variant == 1:
        draw.ellipse((458, 458, 566, 566), fill=rgba("#171a1f"), outline=rgba("#353b44"), width=12)
        draw.ellipse((478, 478, 546, 546), fill=rgba("#f0bb4f"))
    elif variant == 2:
        draw.ellipse((460, 460, 564, 564), fill=rgba("#16181d"), outline=rgba("#4a5059"), width=12)
        draw.ellipse((482, 482, 542, 542), fill=rgba("#f3be55"))
    elif variant == 3:
        draw.ellipse((452, 452, 572, 572), fill=rgba("#15171c"), outline=rgba("#3c424c"), width=12)
        draw.ellipse((478, 478, 546, 546), fill=rgba("#efbb55"))
        draw.line((512, 430, 512, 594), fill=rgba("#1f2329", 140), width=12)
        draw.line((430, 512, 594, 512), fill=rgba("#1f2329", 140), width=12)
    else:
        draw.ellipse((460, 460, 564, 564), fill=rgba("#171a1f"), outline=rgba("#59616b"), width=10)
        draw.ellipse((486, 486, 538, 538), fill=rgba("#fff6d7"))


def compose_badge(variant: int) -> Image.Image:
    img = Image.new("RGBA", (SIZE, SIZE), rgba(BLACK))
    cx, cy = 512, 512

    if variant == 1:
        width_deg = 32
        inner_r = 138
        outer_r = 346
        notch = 58
        waist = 0.78
        tip_pull = 40
        red_map = [True, False, True, False, True, False, True, False]
    elif variant == 2:
        width_deg = 30
        inner_r = 132
        outer_r = 356
        notch = 62
        waist = 0.72
        tip_pull = 34
        red_map = [True, True, False, True, False, False, True, False]
    elif variant == 3:
        width_deg = 33
        inner_r = 142
        outer_r = 340
        notch = 56
        waist = 0.80
        tip_pull = 44
        red_map = [False, True, True, True, False, True, False, False]
    else:
        width_deg = 31
        inner_r = 136
        outer_r = 344
        notch = 58
        waist = 0.76
        tip_pull = 40
        red_map = [True, False, False, True, True, False, False, True]

    angles = [-90, -45, 0, 45, 90, 135, 180, 225]
    for idx, angle in enumerate(angles):
        pts = make_blade_points(cx, cy, inner_r, outer_r, angle, width_deg, notch, waist, tip_pull)
        render_blade(
            img,
            pts,
            red=red_map[idx],
            diagonal=idx % 2 == 0,
            shadow_offset=(18, 16),
        )

    center_hub(img, variant)
    return img


BADGES = [
    ("ref-badge-01", "Closest Crest"),
    ("ref-badge-02", "Sharper Umbra"),
    ("ref-badge-03", "Gold Nexus"),
    ("ref-badge-04", "Cross Seal"),
]


def notes() -> None:
    text = """# Reference Badge Round 2

This round intentionally moves much closer to the supplied reference:
- pure black background
- 8 alternating radial blades
- red / silver metallic treatment
- stronger center hub
- minimal extra structure

Pick order:
- closest to the reference overall: ref-badge-01
- sharpest and cleanest: ref-badge-02
- closest while keeping a premium center: ref-badge-03
- slightly brighter center: ref-badge-04
"""
    (OUT_DIR / "notes.md").write_text(text, encoding="utf-8")


def build_sheet() -> None:
    sheet = Image.new("RGBA", (2440, 1440), rgba("#0a111b"))
    draw = ImageDraw.Draw(sheet)
    title = font(62, bold=True)
    sub = font(30)
    label = font(32)
    draw.text((58, 34), "Xelora reference-style badge round", font=title, fill=rgba("#f6f7fa"))
    draw.text((60, 112), "Closer to the supplied badge in outline, color balance, metallic feel, and center structure.", font=sub, fill=rgba("#9cb0c5"))

    for idx, (name, label_text) in enumerate(BADGES):
        x = 80 + idx * 585
        y = 220
        draw.rounded_rectangle((x, y, x + 500, y + 980), radius=40, fill=rgba("#18263a"))
        img = Image.open(OUT_DIR / f"{name}.png").convert("RGBA").resize((420, 420), Image.Resampling.LANCZOS)
        sheet.alpha_composite(img, (x + 40, y + 74))
        draw.text((x + 30, y + 548), f"{idx + 1}. {label_text}", font=label, fill=rgba("#f4f6fa"))
        draw.text((x + 30, y + 598), name, font=sub, fill=rgba("#94a8be"))

        mini64 = img.resize((96, 96), Image.Resampling.LANCZOS)
        mini32 = img.resize((48, 48), Image.Resampling.LANCZOS).resize((96, 96), Image.Resampling.NEAREST)
        sheet.alpha_composite(mini64, (x + 90, y + 720))
        sheet.alpha_composite(mini32, (x + 270, y + 720))
        draw.text((x + 86, y + 830), "64px", font=sub, fill=rgba("#a6b8cb"))
        draw.text((x + 252, y + 830), "32px x2", font=sub, fill=rgba("#a6b8cb"))
    sheet.save(OUT_DIR / "reference-badge-sheet.png")


def main() -> None:
    ensure_dirs()
    for idx, (name, _label) in enumerate(BADGES, start=1):
        compose_badge(idx).save(OUT_DIR / f"{name}.png")
    build_sheet()
    notes()


if __name__ == "__main__":
    main()
