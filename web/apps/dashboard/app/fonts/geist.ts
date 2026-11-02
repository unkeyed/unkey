import localFont from "next/font/local";

export const GeistSans = localFont({
  src: "./geist-sans-core.woff2",
  weight: "400 900",
  variable: "--font-geist-sans",
  declarations: [
    {
      prop: "font-family",
      value: "GeistSans",
    },
    {
      prop: "unicode-range",
      value:
        "U+0020-007E, U+00A0-00AC, U+00AE-0137, U+0139-013E, U+0141-0148, U+014A-017E, U+01CF-01DC, U+01DE-01E3, U+01EA-01ED, U+01F0, U+01F4-01F5, U+01F8-01FF, U+021E-021F, U+0226-0233, U+0300-0304, U+0306-0308, U+030A-030C, U+0312, U+0326-0328, U+0335, U+0340-0341, U+0344, U+037E, U+0385, U+0387, U+1E02-1E03, U+1E08-1E0B, U+1E10-1E11, U+1E14-1E17, U+1E1C-1E1F, U+1E22-1E23, U+1E26-1E29, U+1E2E-1E31, U+1E3E-1E41, U+1E44-1E45, U+1E4C-1E59, U+1E60-1E61, U+1E64-1E67, U+1E6A-1E6B, U+1E78-1E7D, U+1E86-1E87, U+1E8A-1E91, U+1E97-1E99, U+1EA4-1EA7, U+1EAA-1EAB, U+1EAE-1EB1, U+1EB4-1EB5, U+1EBE-1EC1, U+1EC4-1EC5, U+1ED0-1ED3, U+1ED6-1ED7, U+1FED-1FEF, U+1FFD, U+2000-200A, U+2010-2011, U+2013-2014, U+2018-2019, U+201C-201D, U+2022, U+2026, U+2028-2029, U+202F, U+205F, U+212A-212B, U+21E7",
    },
  ],
});

export const GeistMono = localFont({
  src: "./geist-mono-core.woff2",
  weight: "400 900",
  variable: "--font-geist-mono",
  adjustFontFallback: false,
  fallback: [
    "ui-monospace",
    "SFMono-Regular",
    "Roboto Mono",
    "Menlo",
    "Monaco",
    "Liberation Mono",
    "DejaVu Sans Mono",
    "Courier New",
    "monospace",
  ],
  declarations: [
    {
      prop: "font-family",
      value: "GeistMono",
    },
    {
      prop: "unicode-range",
      value:
        "U+0020-007E, U+00A0-00AC, U+00AE-0131, U+0134-0137, U+0139-013E, U+0141-0148, U+014A-017E, U+01CF-01DC, U+01DE-01E3, U+01EA-01ED, U+01F0, U+01F4-01F5, U+01F8-01FF, U+021E-021F, U+0226-0233, U+0300-0304, U+0306-0308, U+030A-030C, U+0312, U+0326-0328, U+0335, U+0340-0341, U+0344, U+037E, U+0385, U+0387, U+1E02-1E03, U+1E08-1E0B, U+1E10-1E11, U+1E14-1E17, U+1E1C-1E1F, U+1E22-1E23, U+1E26-1E29, U+1E2E-1E31, U+1E3E-1E41, U+1E44-1E45, U+1E4C-1E59, U+1E60-1E61, U+1E64-1E67, U+1E6A-1E6B, U+1E78-1E7D, U+1E86-1E87, U+1E8A-1E91, U+1E97-1E99, U+1EA4-1EA7, U+1EAA-1EAB, U+1EAE-1EB1, U+1EB4-1EB5, U+1EBE-1EC1, U+1EC4-1EC5, U+1ED0-1ED3, U+1ED6-1ED7, U+1FED-1FEF, U+1FFD, U+2000-200A, U+2010-2011, U+2013-2014, U+2018-2019, U+201C-201D, U+2022, U+2026, U+2028-2029, U+202F, U+205F, U+212A-212B, U+21E7",
    },
  ],
});
