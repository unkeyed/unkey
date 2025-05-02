import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { OverviewTooltip } from "@unkey/ui";

export function OverviewTooltipExample() {
  return (
    <div className="flex flex-col gap-6">
      <h3 className="text-lg font-medium p-0 m-0">OverviewTooltip Position Side</h3>
      <RenderComponentWithSnippet>
        <div className="flex flex-col gap-8 p-4">
          <Row>
            <OverviewTooltip content="This is a tooltip on the right">
              Right Tooltip
            </OverviewTooltip>
            <OverviewTooltip content="This is a tooltip on the left" position={{ side: "left" }}>
              Left Tooltip
            </OverviewTooltip>
            <OverviewTooltip content="This is a tooltip on the top" position={{ side: "top" }}>
              Top Tooltip
            </OverviewTooltip>
          </Row>
          <Row>
            <OverviewTooltip
              content="This is a tooltip on the bottom"
              position={{ side: "bottom" }}
            >
              Bottom Tooltip
            </OverviewTooltip>
            <OverviewTooltip
              content="This tooltip has custom alignment"
              position={{
                side: "right",
                align: "start",
                sideOffset: 50,
              }}
            >
              Custom Alignment
            </OverviewTooltip>
            <OverviewTooltip content="This tooltip is disabled" disabled>
              Disabled Tooltip
            </OverviewTooltip>
          </Row>
        </div>
      </RenderComponentWithSnippet>
    </div>
  );
}
