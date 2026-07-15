type MetadataCellProps = {
  label: string;
  children: React.ReactNode;
};

export function MetadataCell({ label, children }: MetadataCellProps) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-xs text-gray-9 font-medium">{label}</span>
      {children}
    </div>
  );
}
