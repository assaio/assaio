#!/usr/bin/env python3
"""Redraw site/og.png -- the card a shared assaio.dev link renders as.

Committed beside the PNG so the asset can be rebuilt rather than re-designed. Palette, mark and
headline are the page's own (site/index.html :root and hero); nothing is invented for the card.

    python3 docs/assets/make-og.py site/og.png
"""
import sys

from PIL import Image, ImageDraw, ImageFont

W, H = 1200, 630
BG = (0xFF, 0xFF, 0xFF)
INK = (0x0F, 0x14, 0x17)
MUTED = (0x4F, 0x5A, 0x64)
FAINT = (0x66, 0x71, 0x7C)
ACCENT = (0x0F, 0x76, 0x6E)
LINE = (0xE2, 0xE6, 0xE9)
TERM = (0x0E, 0x13, 0x16)
TERM_FG = (0xE6, 0xEC, 0xEF)
TERM_DIM = (0x8D, 0x9A, 0xA4)
TERM_ACCENT = (0x5E, 0xEA, 0xD4)

MENLO = "/System/Library/Fonts/Menlo.ttc"
HELV = "/System/Library/Fonts/HelveticaNeue.ttc"
MARGIN = 84  # keeps every glyph inside the area a feed thumbnail crops to

HEADLINE = ["See what your AI coding", "tools cost and produce"]
SUB = "Offline, per-project estimates from local logs, not bills"
TOOLS = "Claude Code · Codex CLI · Gemini CLI · GitHub Copilot CLI · Cline · Antigravity CLI"


def font(path, size, index=0):
    return ImageFont.truetype(path, size, index=index)


def mark(draw, x, y, size):
    """The favicon: a rounded square with a rising stroke."""
    draw.rounded_rectangle([x, y, x + size, y + size], radius=size // 4, fill=ACCENT)
    s = size / 32
    draw.line([(x + 9 * s, y + 22 * s), (x + 16 * s, y + 10 * s), (x + 23 * s, y + 22 * s)],
              fill=BG, width=max(2, round(2.6 * s)), joint="curve")


def main(out):
    img = Image.new("RGB", (W, H), BG)
    d = ImageDraw.Draw(img)

    mark(d, MARGIN, 64, 44)
    d.text((MARGIN + 60, 86), "assaio", font=font(HELV, 34, 1), fill=INK, anchor="lm")
    d.text((W - MARGIN, 86), "assaio.dev", font=font(MENLO, 22), fill=FAINT, anchor="rm")

    head = font(HELV, 70, 1)
    d.text((MARGIN, 222), HEADLINE[0], font=head, fill=INK, anchor="ls")
    d.text((MARGIN, 304), HEADLINE[1], font=head, fill=INK, anchor="ls")
    d.text((MARGIN, 362), SUB, font=font(HELV, 28), fill=MUTED, anchor="ls")

    top = 414
    d.rounded_rectangle([MARGIN, top, W - MARGIN, top + 104], radius=16, fill=TERM)
    mono = font(MENLO, 24)
    d.text((MARGIN + 28, top + 36), "$", font=mono, fill=TERM_DIM, anchor="lm")
    d.text((MARGIN + 54, top + 36), "brew install assaio/tap/assaio-agent", font=mono,
           fill=TERM_FG, anchor="lm")
    d.text((MARGIN + 28, top + 72), "$", font=mono, fill=TERM_DIM, anchor="lm")
    d.text((MARGIN + 54, top + 72), "assaio-agent init", font=mono, fill=TERM_ACCENT, anchor="lm")

    d.line([(MARGIN, 552), (W - MARGIN, 552)], fill=LINE, width=1)
    d.text((MARGIN, 584), TOOLS, font=font(HELV, 21), fill=FAINT, anchor="lm")

    img.save(out, "PNG", optimize=True)
    print(f"wrote {out} {img.size[0]}x{img.size[1]}")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else "og.png")
