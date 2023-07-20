type Props = {
  title: string;
  description?: string;
  actions?: React.ReactNode[];
};

export const PageHeader: React.FC<Props> = ({ title, description, actions }) => {
  return (
    <div className="flex md:items-center flex-col md:flex-row justify-between w-full mb-4 md:mb-8 lg:mb-12">
      <div className="space-y-1">
        <h2 className="text-2xl font-semibold tracking-tight">{title}</h2>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">{description}</p>
      </div>
      <ul className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between md:gap-4">
        {(actions ?? []).map((action, i) => (
          // rome-ignore lint/suspicious/noArrayIndexKey: <explanation>
          <li key={i}>{action}</li>
        ))}
      </ul>
    </div>
  );
};
