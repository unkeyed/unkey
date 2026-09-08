/**
 * Readable text color to place on top of a brand color background.
 *
 * The header, footer, and keys table header paint their background with the
 * customer's `--portal-primary`. A fixed `white` breaks on light brand colors
 * (yellow, pale gray), so choose dark or light text from the background's
 * relative luminance using the WCAG threshold (~0.179) that maximizes contrast
 * against black vs. white.
 *
 * Returns the light color for missing/unparseable input, since the bar then
 * falls back to the dark `--color-gray-12`.
 */
const ON_PRIMARY_LIGHT = "#ffffff";
const ON_PRIMARY_DARK = "#0a0a0a";

export function onPrimaryColor(primaryColor: string | null | undefined): string {
  const rgb = parseHexColor(primaryColor);
  if (!rgb) {
    return ON_PRIMARY_LIGHT;
  }
  return relativeLuminance(rgb) > 0.179 ? ON_PRIMARY_DARK : ON_PRIMARY_LIGHT;
}

type Rgb = { r: number; g: number; b: number };

/** Parse `#rgb` / `#rrggbb` (case-insensitive) into 0–255 channels. */
function parseHexColor(value: string | null | undefined): Rgb | null {
  if (!value) {
    return null;
  }
  const hex = value.trim().replace(/^#/, "");
  const full = hex.length === 3 ? hex.replace(/./g, (c) => c + c) : hex;
  if (!/^[0-9a-fA-F]{6}$/.test(full)) {
    return null;
  }
  return {
    r: Number.parseInt(full.slice(0, 2), 16),
    g: Number.parseInt(full.slice(2, 4), 16),
    b: Number.parseInt(full.slice(4, 6), 16),
  };
}

/** WCAG relative luminance in [0, 1] for an sRGB color. */
function relativeLuminance({ r, g, b }: Rgb): number {
  const [rl, gl, bl] = [r, g, b].map((channel) => {
    const c = channel / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * rl + 0.7152 * gl + 0.0722 * bl;
}
