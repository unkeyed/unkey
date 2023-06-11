type Props = {
  title: string;
  description?: string;
  actions?: React.ReactNode[];
};

export const PageHeader: React.FC<Props> = ({ title, description, actions }) => {
  return (
    <div className="flex items-center justify-between">
      <div className="space-y-1">
        <h2 className="text-2xl font-semibold tracking-tight">{title}</h2>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">{description}</p>
      </div>
      <ul className="flex items-center justify-between gap-4">
        {(actions ?? []).map((action, i) => (
          <li key={i}>{action}</li>
        ))}
      </ul>
    </div>
  );
};
