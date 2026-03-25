/**
 * Standalone preview mode banner for use outside the header.
 * Displays a visual indicator when the session has `preview: true`.
 */
export function PreviewBanner() {
  return (
    <div
      className="bg-warning-3 border-b border-warning-6 px-4 py-1.5 text-center text-sm font-medium text-warning-11"
      role="status"
      aria-label="Preview mode active"
    >
      Preview mode — you are viewing this portal as an end user
    </div>
  );
}
