import { CircleInfo } from "@unkey/icons";

type SettingDescriptionProps = {
  children: React.ReactNode;
};

export const SettingDescription: React.FC<SettingDescriptionProps> = ({ children }) => {
  return (
    <div className="text-[13px] leading-5 max-w-(--setting-w)">
      <output className="text-gray-9 flex gap-2 items-start">
        <CircleInfo iconSize="md-medium" className="shrink-0 mt-[3px]" aria-hidden="true" />
        <span className="flex-1 text-gray-10">{children}</span>
      </output>
    </div>
  );
};
