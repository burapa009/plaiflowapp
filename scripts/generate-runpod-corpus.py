"""Generate synthetic, customer-free PDFs for a 100/500-page OCR pilot."""

import argparse
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter, ImageFont


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--pages", type=int, choices=(100, 500), required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--font", type=Path, default=Path("C:/Windows/Fonts/tahoma.ttf"))
    args = parser.parse_args()
    if not args.font.is_file():
        parser.error("Thai font missing; pass --font with a Thai-capable TTF")
    args.output.mkdir(parents=True, exist_ok=True)
    font = ImageFont.truetype(str(args.font), 38)
    kinds = ("thai_receipt", "thai_tax_invoice", "english_invoice", "rotated_receipt", "low_confidence")
    for group in range(args.pages // 20):
        kind = kinds[group % len(kinds)]
        images, references = [], []
        for page in range(group * 20 + 1, group * 20 + 21):
            high_resolution = kind == "thai_tax_invoice"
            image = Image.new("RGB", (2000, 2800) if high_resolution else (1200, 1700), "white")
            draw = ImageDraw.Draw(image)
            if kind == "english_invoice":
                lines = ("SAMPLE INVOICE - NO CUSTOMER DATA", f"Invoice INV-{page:04d}",
                         "Subtotal 1,000.00", "VAT 70.00", "Total 1,070.00")
            else:
                title = "ใบกำกับภาษี ตัวอย่าง" if kind == "thai_tax_invoice" else "ใบเสร็จรับเงิน ตัวอย่าง"
                lines = (title, f"เลขที่ TEST-{page:04d}", "วันที่ 28/09/2569",
                         "ยอดก่อนภาษี 1,000.00 บาท", "ภาษีมูลค่าเพิ่ม 70.00 บาท", "รวม 1,070.00 บาท")
            for index, line in enumerate(lines):
                draw.text((80, 100 + index * 90), line, font=font,
                          fill="#929292" if kind == "low_confidence" else "black")
            if kind == "rotated_receipt":
                image = image.rotate(5, fillcolor="white")
            if kind == "low_confidence":
                image = image.filter(ImageFilter.GaussianBlur(1.2))
            images.append(image)
            references.extend(lines)
        path = args.output / f"{group + 1:02d}-{kind}-20-pages.pdf"
        images[0].save(path, "PDF", resolution=200, save_all=True, append_images=images[1:])
        path.with_suffix(".pdf.txt").write_text("\n".join(references), encoding="utf-8")
        for image in images:
            image.close()
        print(f"{path.name}: 20 synthetic pages")


if __name__ == "__main__":
    main()
