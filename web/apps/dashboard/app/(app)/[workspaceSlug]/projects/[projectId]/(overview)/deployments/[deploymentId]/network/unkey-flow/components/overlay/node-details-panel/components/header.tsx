import { DoubleChevronRight, type IconProps } from "@unkey/icons";
import type { FC } from "react";
import { CardHeader, type CardHeaderProps } from "../../../nodes/components/card-header";

type Props = {
  onClose: () => void;
  CloseIcon?: FC<IconProps>;
  subSection: CardHeaderProps;
};

export const NodeDetailsPanelHeader = ({
  onClose,
  CloseIcon = DoubleChevronRight,
  subSection,
}: Props) => {
  return (
    <div className="flex items-start justify-between w-full px-3 pt-3 pb-4 gap-3">
      <CardHeader
        type={subSection.type}
        variant="panel"
        icon={subSection.icon}
        title={subSection.title}
        subtitle={subSection.subtitle}
        health={subSection.health}
      />
      <button
        onClick={onClose}
        type="button"
        aria-label="Close details panel"
        className="shrink-0 p-1 rounded-md hover:bg-grayA-3 transition-colors"
      >
        <CloseIcon className="text-gray-9 shrink-0" iconSize="md-regular" />
      </button>
    </div>
  );
};
