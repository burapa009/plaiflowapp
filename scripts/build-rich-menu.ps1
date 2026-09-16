param([string] $Output = "web/public/rich-menu.png")
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing
$bitmap = [Drawing.Bitmap]::new(2500, 1686)
$graphics = [Drawing.Graphics]::FromImage($bitmap)
$graphics.SmoothingMode = [Drawing.Drawing2D.SmoothingMode]::AntiAlias
$graphics.TextRenderingHint = [Drawing.Text.TextRenderingHint]::AntiAliasGridFit
$background = [Drawing.SolidBrush]::new([Drawing.Color]::FromArgb(244, 251, 249))
$primary = [Drawing.SolidBrush]::new([Drawing.Color]::FromArgb(16, 132, 115))
$ink = [Drawing.SolidBrush]::new([Drawing.Color]::FromArgb(22, 45, 43))
$muted = [Drawing.SolidBrush]::new([Drawing.Color]::FromArgb(87, 111, 108))
$border = [Drawing.Pen]::new([Drawing.Color]::FromArgb(202, 222, 218), 4)
$title = [Drawing.Font]::new('Leelawadee UI', 66, [Drawing.FontStyle]::Bold)
$subtitle = [Drawing.Font]::new('Leelawadee UI', 34, [Drawing.FontStyle]::Regular)
$graphics.FillRectangle($background, 0, 0, 2500, 1686)
function From-Base64([string] $Value) { [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($Value)) }
$labels = @('4Lig4Liy4Lie4Lij4Lin4Lih', '4LiH4Liy4LiZ', '4LiE4Li54LmI4LiE4LmJ4Liy', '4LiZ4Liz4LmA4LiC4LmJ4Liy', '4Liq4LmI4LiH4Lit4Lit4LiB', '4LiV4Lix4LmJ4LiH4LiE4LmI4Liy') | ForEach-Object { From-Base64 $_ }
$descriptions = @('4Liq4LiW4Liy4LiZ4Liw4LiY4Li44Lij4LiB4Li04LiI', '4LiV4Li04LiU4LiV4Liy4Lih4LiH4Liy4LiZ4LiX4Li14Lih', '4LiC4LmJ4Lit4Lih4Li54Lil4Lic4Li54LmJ4LiC4Liy4Lii', 'Q1NWIC8gWExTWA==', '4LmE4Lif4Lil4LmM4Lij4Liy4Lii4LiH4Liy4LiZ', 'Q29ubmVjdGlvbnM=') | ForEach-Object { From-Base64 $_ }
for ($index = 0; $index -lt 6; $index++) {
    $column = $index % 3
    $row = [Math]::Floor($index / 3)
    $x = $column * 833
    $y = $row * 843
    $graphics.DrawRectangle($border, $x + 24, $y + 24, 785, 795)
    $graphics.FillEllipse($primary, $x + 337, $y + 185, 160, 160)
    $number = [Drawing.Font]::new('Segoe UI', 52, [Drawing.FontStyle]::Bold)
    $format = [Drawing.StringFormat]::new(); $format.Alignment = 'Center'; $format.LineAlignment = 'Center'
    $graphics.DrawString(($index + 1).ToString(), $number, [Drawing.Brushes]::White, [Drawing.RectangleF]::new($x + 337, $y + 185, 160, 160), $format)
    $graphics.DrawString($labels[$index], $title, $ink, [Drawing.RectangleF]::new($x + 40, $y + 420, 753, 110), $format)
    $graphics.DrawString($descriptions[$index], $subtitle, $muted, [Drawing.RectangleF]::new($x + 40, $y + 545, 753, 80), $format)
    $number.Dispose(); $format.Dispose()
}
$directory = Split-Path -Parent $Output
if ($directory) { New-Item -ItemType Directory -Force -Path $directory | Out-Null }
$bitmap.Save($Output, [Drawing.Imaging.ImageFormat]::Png)
$title.Dispose(); $subtitle.Dispose(); $border.Dispose(); $background.Dispose(); $primary.Dispose(); $ink.Dispose(); $muted.Dispose(); $graphics.Dispose(); $bitmap.Dispose()
