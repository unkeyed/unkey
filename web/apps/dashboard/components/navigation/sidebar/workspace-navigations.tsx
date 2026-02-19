import { cn } from "../../../lib/utils";

export type NavItem = {
  disabled?: boolean;
  tooltip?: string;
  icon: React.ElementType | null;
  href: string;
  external?: boolean;
  label: string | React.ReactNode;
  active?: boolean;
  tag?: React.ReactNode;
  hidden?: boolean;
  items?: NavItem[];
  loadMoreAction?: boolean;
  showSubItems?: boolean;
};

const Tag: React.FC<{ label: string; className?: string }> = ({ label, className }) => (
  <div
    className={cn(
      "border text-gray-11 border-gray-6 hover:border-gray-8 rounded text-xs px-1 py-0.5 font-mono",
      className,
    )}
  >
    {label}
  </div>
);

export { Tag };
