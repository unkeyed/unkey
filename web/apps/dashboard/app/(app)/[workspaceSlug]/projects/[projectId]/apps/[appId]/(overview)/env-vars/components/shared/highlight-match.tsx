type HighlightMatchProps = {
  text: string;
  query: string;
};

function escapeRegExp(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

export function HighlightMatch({ text, query }: HighlightMatchProps) {
  if (!query) {
    return <>{text}</>;
  }

  const parts = text.split(new RegExp(`(${escapeRegExp(query)})`, "gi"));

  return (
    <>
      {parts.map((part, i) =>
        part.toLowerCase() === query.toLowerCase() ? (
          // biome-ignore lint/suspicious/noArrayIndexKey: parts are derived from splitting text by query — stable order, no reordering
          <mark key={i} className="bg-yellow-4 text-yellow-12 rounded-sm">
            {part}
          </mark>
        ) : (
          part
        ),
      )}
    </>
  );
}
