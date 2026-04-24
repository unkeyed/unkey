/**
 * Preview mode banner displayed when the session has `preview: true`.
 */
export function PreviewBanner() {
  return (
    <output
      className="block border-warning-6 border-b bg-warning-3 px-4 py-1.5 text-center font-medium text-sm text-warning-11"
      aria-label="Preview mode active"
    >
      Preview mode — you are viewing this portal as an end user
    </output>
  );
}
