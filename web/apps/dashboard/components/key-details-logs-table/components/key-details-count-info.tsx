type Props = {
  visibleCount: number;
  totalCount: number;
};

export const KeyDetailsCountInfo = ({ visibleCount, totalCount }: Props) => {
  return (
    <div className="flex gap-2">
      <span>Showing</span>{" "}
      <span className="text-gray-12">{new Intl.NumberFormat().format(visibleCount)}</span>
      <span>of</span>
      {new Intl.NumberFormat().format(totalCount)}
      <span>requests</span>
    </div>
  );
};
