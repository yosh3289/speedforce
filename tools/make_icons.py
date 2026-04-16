"""Icon generator: draws flat-color-icons flash bolt → multi-size ICO.

Uses the same polygon coordinates as Icons8 flat-color-icons flash_on.
No external SVG renderer needed — pure Pillow.
"""

import os
import shutil
from PIL import Image, ImageDraw

BASE = "E:/0-syb/dev/speedforce"
OUT_DIR = os.path.join(BASE, "assets/icons")
EMBED_DIR = os.path.join(BASE, "internal/ui/tray/icons")

VARIANTS = {
    "blue": (41, 121, 255),
    "yellow": (255, 196, 0),
    "red": (213, 0, 0),
}

RENDER_SIZE = 256
ICO_SIZES = [(16, 16), (24, 24), (32, 32), (48, 48)]


def draw_lightning(color: tuple, size: int) -> Image.Image:
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    s = size / 48.0
    points = [
        (33 * s, 22 * s),
        (23.6 * s, 22 * s),
        (30 * s, 5 * s),
        (19 * s, 5 * s),
        (13 * s, 26 * s),
        (21.6 * s, 26 * s),
        (17 * s, 45 * s),
    ]
    draw.polygon(points, fill=color)
    return img


def main() -> None:
    os.makedirs(OUT_DIR, exist_ok=True)
    os.makedirs(EMBED_DIR, exist_ok=True)

    for name, color in VARIANTS.items():
        big = draw_lightning(color, RENDER_SIZE)

        ico_name = f"lightning-{name}.ico"
        ico_path = os.path.join(OUT_DIR, ico_name)
        big.save(ico_path, format="ICO", sizes=ICO_SIZES)
        sz = os.path.getsize(ico_path)
        print(f"wrote {ico_path} ({sz} bytes)")

        embed_path = os.path.join(EMBED_DIR, ico_name)
        shutil.copy2(ico_path, embed_path)
        print(f"  → {embed_path}")


if __name__ == "__main__":
    main()
