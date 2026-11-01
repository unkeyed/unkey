# Geist font subsets

The dashboard loads Geist and Geist Mono from these files, not from the
`geist` package directly. Each font is cut into two files from
`node_modules/geist/dist/fonts/*/Geist*-Variable.woff2`:

- `*-core.woff2`: preloaded on every page through `geist.ts`.
- `*-rest.woff2`: declared in `geist.css` and downloaded only when a page
  renders a character in its `unicode-range`.

There is no generator script. Rebuild the files by hand (fontTools works) when:

- the `geist` package version changes,
- the UI uses a weight below 400,
- the UI turns on an OpenType feature that is not listed below, or sets a
  `lang` other than English,
- a character outside the core starts to show on most pages.

## Rules

Rendering must stay pixel-identical to the full font. Each rule below fixes a
regression that a pixel diff found.

1. Core characters: U+0020-007E, U+00A0-017F, U+0300-036F, U+2013, U+2014,
   U+2018, U+2019, U+201C, U+201D, U+2022, U+2026, U+21E7. The rest file gets
   every other character in the source font.
2. Limit the `wght` axis to 400-900, not 400-700. A top limit below the 900
   master makes the instancer round deltas, and weights 500-700 render
   differently. Set `font-weight: 400 900` on every face.
3. Keep only the `ccmp`, `kern`, `liga`, `mark`, `mkmk` and `tnum` features.
   Geist defines `locl` only for non-English languages. Drop all hinting.
4. Add these to the core `unicode-range` even though the core cmap does not
   contain them: U+2000-200A, U+2010, U+2011, U+2028, U+2029, U+202F, U+205F.
   Browsers draw them from Geist's space and hyphen glyphs. Chrome puts U+202F
   before AM/PM in formatted times.
5. A character that Geist lacks, but whose canonical decomposition (NFD) Geist
   has in full, goes in the range of the file that holds its base letter.
   Copy the marks it needs into that file and into that file's range. Chrome
   shapes only with characters inside a face's range.
6. Do not stamp a new `head.modified` timestamp, so the output is
   reproducible.
7. Keep the family names `GeistSans` and `GeistMono` and the `localFont`
   options in `geist.ts`. The generated `GeistSans Fallback` face depends on
   them.

## Verify

- Every original codepoint is in some range. Ranges overlap only on
  combining marks.
- Outlines and advance widths match the source font at weights 400-900.
- Pixel-diff dashboard pages against `main`, including text with accents,
  Cyrillic and CJK.
