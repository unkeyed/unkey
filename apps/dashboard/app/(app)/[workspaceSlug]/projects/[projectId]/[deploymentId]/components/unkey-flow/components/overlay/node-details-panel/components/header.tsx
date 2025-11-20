import { Book2, DoubleChevronRight, type IconProps } from "@unkey/icons";
import type { FC } from "react";
import { CardHeader, type CardHeaderProps } from "../../../nodes/deploy-node";

type Props = {
  onClose: () => void;
  title?: string;
  CloseIcon?: FC<IconProps>;
  Icon?: FC<IconProps>;
  subSection: CardHeaderProps;
};

export const NodeDetailsPanelHeader = ({
  onClose,
  title = "Details",
  Icon = Book2,
  CloseIcon = DoubleChevronRight,
  subSection,
}: Props) => {
  return (
    <>
      <div className="flex items-center justify-between h-12 border-b border-grayA-4 w-full px-3 py-2.5">
        <div className="flex gap-2.5 items-center p-2 border rounded-lg border-grayA-5 bg-grayA-2 h-[26px]">
          <Icon className="text-gray-12" iconSize="sm-regular" />
          <span className="text-accent-12 font-medium text-[13px] leading-4">{title}</span>
        </div>
        <button onClick={onClose} type="button">
          <CloseIcon className="text-gray-8 shrink-0" iconSize="lg-regular" />
        </button>
      </div>
      <div className="flex items-center justify-between w-full px-3 py-4">
        <CardHeader
          variant="panel"
          icon={subSection.icon}
          title={subSection.title}
          subtitle={subSection.subtitle}
          health={subSection.health}
        />
      </div>
    </>
  );
};
