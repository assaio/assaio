#!/usr/bin/env python3
"""Redraw site/og.png -- the card a shared assaio.dev link renders as.

Committed beside the PNG so the asset can be rebuilt rather than re-designed, which is the
same reason every other number here lives in one place. Palette and type are the page's own
(site/index.html :root); nothing is invented for the card.

    python3 docs/assets/make-og.py site/og.png
"""
import sys

from PIL import Image, ImageDraw, ImageFont

W, H = 1200, 630
PAPER = (0xEF, 0xEA, 0xE1)
INK = (0x1A, 0x17, 0x13)
DIM = (0x6B, 0x64, 0x59)
FAINT = (0x8D, 0x85, 0x7A)
TEAL = (0x3E, 0x8B, 0x79)
RULE = (0xD5, 0xCF, 0xC4)
WATERMARK = (0xE2, 0xDC, 0xD1)

MENLO = "/System/Library/Fonts/Menlo.ttc"
HELV = "/System/Library/Fonts/HelveticaNeue.ttc"
MARGIN = 84  # keeps every glyph inside the area a feed thumbnail crops to


def font(path, size, index=0):
    return ImageFont.truetype(path, size, index=index)


def hexagon(draw, cx, cy, r, colour, width):
    """The favicon's mark: a flat-sided hexagon around a filled centre."""
    pts = [
        (cx, cy - r),
        (cx + r * 0.866, cy - r * 0.5),
        (cx + r * 0.866, cy + r * 0.5),
        (cx, cy + r),
        (cx - r * 0.866, cy + r * 0.5),
        (cx - r * 0.866, cy - r * 0.5),
    ]
    draw.polygon(pts, outline=colour, width=width)
    draw.ellipse([cx - r * 0.27, cy - r * 0.27, cx + r * 0.27, cy + r * 0.27], fill=colour)


def main(out):
    img = Image.new("RGB", (W, H), PAPER)
    d = ImageDraw.Draw(img)

    # The mark again, oversized and quiet, so the card is recognisable at feed scale without
    # anything competing with the question.
    hexagon(d, 1035, 300, 210, WATERMARK, 26)

    hexagon(d, MARGIN + 21, 92, 21, TEAL, 3)
    d.text((MARGIN + 56, 92), "assaio", font=font(MENLO, 30, 1), fill=INK, anchor="lm")

    d.text((MARGIN, 214), "Is your AI coding", font=font(HELV, 78, 1), fill=INK, anchor="ls")
    d.text((MARGIN, 298), "spend delivering?", font=font(HELV, 78, 1), fill=INK, anchor="ls")

    d.text((MARGIN, 372), "$ per 100 AI lines, per project.",
           font=font(MENLO, 28), fill=TEAL, anchor="ls")

    sub = font(HELV, 26)
    d.text((MARGIN, 428), "Offline-first analytics for Claude Code, Codex CLI, Gemini CLI,",
           font=sub, fill=DIM, anchor="ls")
    d.text((MARGIN, 464), "GitHub Copilot CLI and Cline. No account, no upload.",
           font=sub, fill=DIM, anchor="ls")

    d.line([(MARGIN, 528), (W - MARGIN, 528)], fill=RULE, width=1)
    foot = font(MENLO, 21)
    d.text((MARGIN, 566), "assaio.dev", font=foot, fill=INK, anchor="lm")
    d.text((W - MARGIN, 566), "no telemetry  ·  prompts are never read",
           font=foot, fill=FAINT, anchor="rm")

    img.save(out, "PNG", optimize=True)
    print(f"wrote {out} {img.size[0]}x{img.size[1]}")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else "og.png")
