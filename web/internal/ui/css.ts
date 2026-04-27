import "./src/tailwind.css";

/**
 * Loads color CSS variables via JS import.
 *
 * For Tailwind v4 apps, prefer `@import "@unkey/ui/src/tailwind.css"` (or the
 * relative path) in your main Tailwind CSS file instead — this ensures @theme
 * blocks are processed and utility classes (bg-gray-1, text-error-9, etc.) are
 * generated.
 *
 * The JS import path still works for loading raw CSS custom properties
 * but will NOT generate Tailwind utility classes.
 */
