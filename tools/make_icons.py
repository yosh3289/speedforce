"""One-off icon generator.

Takes the source pixel-art lightning PNG and emits three tinted .ico
files (blue / yellow / red) for use by the tray app.
"""

import os
from PIL import Image

SRC = "E:/0-syb/dev/speedforce/Pixel_Flash_Icon.png"
OUT_DIR = "E:/0-syb/dev/speedforce/assets/icons"

VARIANTS = {
    "blue":   ((41, 121, 255),  (13, 71, 161)),
    "yellow": ((255, 196, 0),   (180, 100, 0)),
    "red":    ((213, 0, 0),     (127, 0, 0)),
}

BRIGHT_THRESHOLD = 150
ALPHA_THRESHOLD = 50


def main() -> None:
    os.makedirs(OUT_DIR, exist_ok=True)

    src = Image.open(SRC).convert("RGBA")
    bbox = src.getbbox()
    if bbox:
        src = src.crop(bbox)

    w, h = src.size
    side = max(w, h)
    padded = Image.new("RGBA", (side, side), (0, 0, 0, 0))
    padded.paste(src, ((side - w) // 2, (side - h) // 2))
    src = padded

    src_pixels = src.load()

    for name, (primary, outline) in VARIANTS.items():
        out = Image.new("RGBA", src.size, (0, 0, 0, 0))
        dst_pixels = out.load()

        for x in range(src.width):
            for y in range(src.height):
                r, g, b, a = src_pixels[x, y]
                if a < ALPHA_THRESHOLD:
                    continue
                brightness = (r + g + b) / 3
                color = primary if brightness > BRIGHT_THRESHOLD else outline
                dst_pixels[x, y] = color + (a,)

        out_path = os.path.join(OUT_DIR, f"lightning-{name}.ico")
        out.save(
            out_path,
            format="ICO",
            sizes=[(16, 16), (24, 24), (32, 32), (48, 48)],
        )
        print(f"wrote {out_path}")


if __name__ == "__main__":
    main()
