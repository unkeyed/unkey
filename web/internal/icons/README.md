# @unkey/icons

All icons come from the Nucleo UI set. The team holds an extended Nucleo
license. Do not add icons from other sets.

## Which family to use

- Outline, 18px is the default. Use it unless you have a reason not to.
- Outline, 12px is for glyphs that render at `size-3` or smaller.
- Fill, 18px is only for a filled glyph that the design needs.

## How to add an icon

1. Export the SVG from the Nucleo app.
2. Create `src/icons/<name>-outline-18.tsx`.
3. Follow the example below. Keep `stroke="currentColor"`, keep
   `strokeWidth={1.5}` and keep the `viewBox`.
4. Name the export `Icon<Name>Outline18`.
5. Add the file to `src/index.ts`. Keep that list sorted.

## Example

```tsx
import type { IconProps } from "../props";

export function IconPlusOutline18(props: IconProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={18}
      height={18}
      viewBox="0 0 18 18"
      {...props}
    >
      <line
        x1="9"
        y1="3.25"
        x2="9"
        y2="14.75"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
      />
      <line
        x1="3.25"
        y1="9"
        x2="14.75"
        y2="9"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
      />
    </svg>
  );
}
```

## Rules for call sites

- Do not set `strokeWidth`, `size`, `width` or `height`.
- Do not use `h-` or `w-` classes.
- Set the size with one `size-*` class.
- Do not scale hero icons yourself. `Empty.Icon` and `EmptyHero.Icons` do it.
