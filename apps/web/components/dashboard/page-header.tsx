type Props = {
  title: string;
  description?: string;
  actions?: React.ReactNode[];
};

export const PageHeader: React.FC<Props> = ({ title, description, actions }) => {
  const actionRows: React.ReactNode[][] = [];
  if (actions) {
    for (let i = 0; i < actions.length; i += 3) {
      actionRows.push(actions.slice(i, i + 3));
    }
  }

  return (
    <div className="flex flex-col items-start justify-between w-full gap-2 mb-4 md:items-center md:flex-row md:gap-4">
      <div className="space-y-1 ">
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400">{description}</p>
      </div>
      {actionRows.map((row, i) => (
        <ul
          key={i.toString()}
          className="flex flex-wrap items-center justify-end gap-2 md:gap-4 md:flex-nowrap"
        >
          {row.map((action, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
            <li key={i}>{action}</li>
          ))}
        </ul>
      ))}
    </div>
  );
};
