from __future__ import annotations

from pathlib import Path
from PIL import Image, ImageDraw, ImageFilter, ImageFont


SIZE = 1024
RADIUS = 224
OUT_DIR = Path(__file__).resolve().parent
IOS_DIR = OUT_DIR / "ios-1024"
WEB64_DIR = OUT_DIR / "web-64"
WEB32_DIR = OUT_DIR / "web-32"

BG = "#07111f"
BG_ALT = "#0b1727"
INK = "#e8f3ff"
CYAN = "#53d9ff"
TEAL = "#1fbfa5"
BLUE = "#2f74ff"
GOLD = "#f2bf45"
SLATE = "#1b2a3d"


def ensure_dirs() -> None:
    for path in (OUT_DIR, IOS_DIR, WEB64_DIR, WEB32_DIR):
        path.mkdir(parents=True, exist_ok=True)


def hex_to_rgba(value: str, alpha: int = 255) -> tuple[int, int, int, int]:
    value = value.lstrip("#")
    return tuple(int(value[i : i + 2], 16) for i in (0, 2, 4)) + (alpha,)


def new_canvas() -> Image.Image:
    img = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    draw.rounded_rectangle(
        (48, 48, SIZE - 48, SIZE - 48),
        radius=RADIUS,
        fill=hex_to_rgba(BG),
    )
    gradient = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    gdraw = ImageDraw.Draw(gradient)
    for i in range(10):
        inset = 96 + (i * 18)
        alpha = max(6, 32 - (i * 2))
        color = CYAN if i % 2 == 0 else BLUE
        gdraw.rounded_rectangle(
            (inset, inset, SIZE - inset, SIZE - inset),
            radius=max(64, RADIUS - i * 12),
            outline=hex_to_rgba(color, alpha),
            width=6,
        )
    gradient = gradient.filter(ImageFilter.GaussianBlur(10))
    img.alpha_composite(gradient)
    return img


def add_glow(base: Image.Image, shape: tuple[int, int, int, int], color: str, blur: int) -> None:
    glow = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    gdraw = ImageDraw.Draw(glow)
    gdraw.ellipse(shape, fill=hex_to_rgba(color, 105))
    glow = glow.filter(ImageFilter.GaussianBlur(blur))
    base.alpha_composite(glow)


def draw_band(draw: ImageDraw.ImageDraw, points: list[tuple[int, int]], fill: str) -> None:
    draw.polygon(points, fill=hex_to_rgba(fill))


def draw_option_01() -> Image.Image:
    img = new_canvas()
    add_glow(img, (300, 300, 724, 724), CYAN, 72)
    draw = ImageDraw.Draw(img)
    draw_band(draw, [(250, 180), (382, 180), (774, 844), (642, 844)], CYAN)
    draw_band(draw, [(642, 180), (774, 180), (382, 844), (250, 844)], INK)
    draw.polygon(
        [(512, 398), (640, 512), (512, 626), (384, 512)],
        fill=hex_to_rgba(BG_ALT),
    )
    draw.polygon(
        [(512, 430), (602, 512), (512, 594), (422, 512)],
        fill=hex_to_rgba(GOLD),
    )
    return img


def draw_option_02() -> Image.Image:
    img = new_canvas()
    add_glow(img, (240, 240, 784, 784), TEAL, 88)
    draw = ImageDraw.Draw(img)
    draw.polygon(
        [(198, 270), (330, 182), (512, 438), (694, 182), (826, 270), (584, 512), (826, 754), (694, 842), (512, 586), (330, 842), (198, 754), (440, 512)],
        fill=hex_to_rgba(INK),
    )
    draw.polygon(
        [(278, 310), (368, 250), (512, 460), (656, 250), (746, 310), (562, 512), (746, 714), (656, 774), (512, 564), (368, 774), (278, 714), (462, 512)],
        fill=hex_to_rgba(SLATE),
    )
    draw.polygon(
        [(352, 348), (430, 296), (512, 428), (594, 296), (672, 348), (548, 512), (672, 676), (594, 728), (512, 596), (430, 728), (352, 676), (476, 512)],
        fill=hex_to_rgba(BLUE),
    )
    draw.polygon(
        [(512, 446), (576, 512), (512, 578), (448, 512)],
        fill=hex_to_rgba(CYAN),
    )
    return img


def draw_option_03() -> Image.Image:
    img = new_canvas()
    add_glow(img, (246, 246, 778, 778), BLUE, 80)
    draw = ImageDraw.Draw(img)
    draw.arc((180, 180, 844, 844), start=28, end=152, fill=hex_to_rgba(CYAN), width=40)
    draw.arc((180, 180, 844, 844), start=208, end=332, fill=hex_to_rgba(TEAL), width=24)
    draw_band(draw, [(292, 188), (406, 188), (732, 836), (618, 836)], INK)
    draw_band(draw, [(618, 188), (732, 188), (406, 836), (292, 836)], CYAN)
    draw.ellipse((438, 438, 586, 586), fill=hex_to_rgba(BG_ALT))
    draw.ellipse((468, 468, 556, 556), fill=hex_to_rgba(GOLD))
    return img


def draw_option_04() -> Image.Image:
    img = new_canvas()
    add_glow(img, (256, 256, 768, 768), CYAN, 66)
    draw = ImageDraw.Draw(img)
    left = [(256, 202), (410, 202), (548, 430), (444, 522), (256, 202)]
    left_bottom = [(444, 522), (548, 430), (742, 822), (590, 822)]
    right = [(768, 202), (616, 202), (478, 430), (582, 522), (768, 202)]
    right_bottom = [(582, 522), (478, 430), (282, 822), (436, 822)]
    draw.polygon(left, fill=hex_to_rgba(INK))
    draw.polygon(left_bottom, fill=hex_to_rgba(CYAN))
    draw.polygon(right, fill=hex_to_rgba(TEAL))
    draw.polygon(right_bottom, fill=hex_to_rgba(INK))
    draw.line((404, 214, 534, 425), fill=hex_to_rgba("#ffffff", 90), width=12)
    draw.line((620, 214, 492, 425), fill=hex_to_rgba("#ffffff", 70), width=10)
    draw.polygon([(512, 450), (582, 512), (512, 574), (442, 512)], fill=hex_to_rgba(BG_ALT))
    return img


def draw_option_05() -> Image.Image:
    img = new_canvas()
    add_glow(img, (300, 220, 724, 644), GOLD, 90)
    draw = ImageDraw.Draw(img)
    draw_band(draw, [(272, 208), (396, 208), (728, 816), (604, 816)], CYAN)
    draw_band(draw, [(604, 208), (728, 208), (396, 816), (272, 816)], INK)
    draw.ellipse((430, 430, 594, 594), fill=hex_to_rgba(BG_ALT))
    draw.ellipse((458, 458, 566, 566), fill=hex_to_rgba(GOLD))
    draw.line((512, 186, 512, 356), fill=hex_to_rgba("#ffffff", 110), width=24)
    draw.line((512, 186, 512, 280), fill=hex_to_rgba(CYAN, 180), width=12)
    return img


def draw_option_06() -> Image.Image:
    img = new_canvas()
    add_glow(img, (250, 250, 774, 774), TEAL, 78)
    draw = ImageDraw.Draw(img)
    draw.polygon([(214, 264), (360, 176), (560, 456), (476, 540)], fill=hex_to_rgba(INK))
    draw.polygon([(664, 176), (810, 264), (548, 540), (464, 456)], fill=hex_to_rgba(CYAN))
    draw.polygon([(476, 484), (560, 568), (360, 848), (214, 760)], fill=hex_to_rgba(TEAL))
    draw.polygon([(464, 568), (548, 484), (810, 760), (664, 848)], fill=hex_to_rgba(INK))
    draw.rounded_rectangle((392, 392, 632, 632), radius=72, fill=hex_to_rgba(BG_ALT))
    draw.rounded_rectangle((442, 442, 582, 582), radius=40, fill=hex_to_rgba(BLUE))
    draw.rounded_rectangle((476, 476, 548, 548), radius=24, fill=hex_to_rgba(INK))
    return img


OPTIONS = [
    ("option-01", "Apex Core", draw_option_01),
    ("option-02", "Prism Logic", draw_option_02),
    ("option-03", "Orbit Reason", draw_option_03),
    ("option-04", "Folded Engine", draw_option_04),
    ("option-05", "Beacon Core", draw_option_05),
    ("option-06", "Nexus Kernel", draw_option_06),
]


def save_outputs(name: str, img: Image.Image) -> None:
    img.save(IOS_DIR / f"{name}.png")
    img.resize((64, 64), Image.Resampling.LANCZOS).save(WEB64_DIR / f"{name}.png")
    img.resize((32, 32), Image.Resampling.LANCZOS).save(WEB32_DIR / f"{name}.png")
    img.save(OUT_DIR / f"{name}.png")


def load_font(size: int) -> ImageFont.ImageFont:
    candidates = [
        Path("C:/Windows/Fonts/segoeui.ttf"),
        Path("C:/Windows/Fonts/arial.ttf"),
    ]
    for path in candidates:
        if path.exists():
            return ImageFont.truetype(str(path), size=size)
    return ImageFont.load_default()


def build_contact_sheet() -> None:
    card_w = 600
    card_h = 660
    margin = 48
    cols = 2
    rows = 3
    sheet = Image.new("RGBA", (cols * card_w + margin * 3, rows * card_h + margin * 4), hex_to_rgba("#08111d"))
    draw = ImageDraw.Draw(sheet)
    title_font = load_font(44)
    label_font = load_font(28)
    note_font = load_font(22)
    draw.text((margin, 24), "Xelora icon candidates", font=title_font, fill=hex_to_rgba("#f4f7fb"))
    draw.text((margin, 84), "Direction: enterprise reasoning platform, X-forward core mark", font=note_font, fill=hex_to_rgba("#90a4bc"))
    for idx, (name, label, _) in enumerate(OPTIONS):
        row = idx // cols
        col = idx % cols
        x = margin + col * (card_w + margin)
        y = 140 + row * (card_h + margin)
        draw.rounded_rectangle((x, y, x + card_w, y + card_h), radius=42, fill=hex_to_rgba("#0f1b2c"))
        icon = Image.open(OUT_DIR / f"{name}.png").convert("RGBA").resize((460, 460), Image.Resampling.LANCZOS)
        sheet.alpha_composite(icon, (x + 70, y + 46))
        draw.text((x + 36, y + 538), f"{idx + 1}. {label}", font=label_font, fill=hex_to_rgba("#edf5ff"))
        draw.text((x + 36, y + 580), name, font=note_font, fill=hex_to_rgba("#89a0bb"))
    sheet.save(OUT_DIR / "contact-sheet.png")


def build_readability_sheet() -> None:
    sheet = Image.new("RGBA", (1440, 980), hex_to_rgba("#08111d"))
    draw = ImageDraw.Draw(sheet)
    title_font = load_font(42)
    label_font = load_font(24)
    draw.text((40, 24), "Xelora favicon readability check", font=title_font, fill=hex_to_rgba("#f4f7fb"))
    draw.text((40, 78), "64px and 32px previews to see whether the X silhouette survives reduction", font=label_font, fill=hex_to_rgba("#90a4bc"))
    for idx, (name, label, _) in enumerate(OPTIONS):
        y = 150 + idx * 128
        draw.rounded_rectangle((36, y, 1404, y + 104), radius=24, fill=hex_to_rgba("#0f1b2c"))
        draw.text((72, y + 34), f"{idx + 1}. {label}", font=label_font, fill=hex_to_rgba("#edf5ff"))
        icon64 = Image.open(WEB64_DIR / f"{name}.png").convert("RGBA")
        icon32 = Image.open(WEB32_DIR / f"{name}.png").convert("RGBA")
        sheet.alpha_composite(icon64, (900, y + 20))
        sheet.alpha_composite(icon32.resize((64, 64), Image.Resampling.NEAREST), (1050, y + 20))
        draw.text((980, y + 74), "64px", font=label_font, fill=hex_to_rgba("#89a0bb"))
        draw.text((1112, y + 74), "32px x2", font=label_font, fill=hex_to_rgba("#89a0bb"))
    sheet.save(OUT_DIR / "favicon-readability-sheet.png")


def write_docs() -> None:
    prompts = """# Xelora Icon Direction

- Product name: Xelora
- Brand type: enterprise knowledge understanding and reasoning platform
- Core requirement: make the X the dominant memory point
- Visual metaphor: reasoning core, knowledge engine, intelligent nucleus
- Asset goals: app icon + website favicon, readable at 32px, no text inside the icon
- Style: premium, restrained, platform-grade, dark base with cool highlights
"""
    (OUT_DIR / "prompts.md").write_text(prompts, encoding="utf-8")

    choices = """# Xelora Candidate Notes

1. Apex Core
   Cleanest X silhouette. Strong central diamond and best balance between enterprise tone and recall.

2. Prism Logic
   More constructed and layered. Feels like reasoning through stacked logic planes.

3. Orbit Reason
   Adds subtle orbit motion around the X. Feels more dynamic and systems-oriented.

4. Folded Engine
   Ribbon-like X with directional energy. Slightly more product-brand than infrastructure-brand.

5. Beacon Core
   Strongest emphasis on a lit reasoning core. Distinct and memorable, but a little more dramatic.

6. Nexus Kernel
   Most “platform kernel” feeling. Technical and modular, with a strong central square core.

Recommendation:
- First look: option-01, option-05, option-06
- Most balanced default: option-01
- Strongest “X as identity”: option-05
- Most platform/engine feeling: option-06
"""
    (OUT_DIR / "choices.md").write_text(choices, encoding="utf-8")


def main() -> None:
    ensure_dirs()
    for name, _label, builder in OPTIONS:
        save_outputs(name, builder())
    build_contact_sheet()
    build_readability_sheet()
    write_docs()


if __name__ == "__main__":
    main()
