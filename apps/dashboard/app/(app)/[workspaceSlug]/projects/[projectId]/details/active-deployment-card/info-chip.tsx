import type { IconProps } from "@unkey/icons/src/props";

type InfoChipProps = {
  icon: React.ComponentType<IconProps>;
  children: React.ReactNode;
};

export const InfoChip = ({ icon: Icon, children }: InfoChipProps) => (
  <div className="gap-2 flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3 p-1.5 h-[22px] rounded-md">
    <Icon iconSize="md-medium" className="text-gray-12" />
    {children}
  </div>
);
