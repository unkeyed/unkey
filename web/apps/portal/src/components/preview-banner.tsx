/**
 * Preview mode banner displayed when the session has `preview: true`.
 */
export function PreviewBanner() {
  return (
    <output
      className="block bg-warning-3 border-b border-warning-6 px-4 py-1.5 text-center text-sm font-medium text-warning-11"
      aria-label="Preview mode active"
    >
      Preview mode — you are viewing this portal as an end user
    </output>
  );
}
