import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { InfoTooltip } from "@unkey/ui";

export function InfoTooltipExample() {
  return (
    <div className="flex flex-col gap-6">
      <h3 className="text-lg font-medium p-0 m-0">InfoTooltip Position Side</h3>
      <RenderComponentWithSnippet
        customCodeSnippet={`<div className="flex flex-col gap-8 p-4">
    <Row>
      <InfoTooltip content="This is a tooltip on the right">Right Tooltip</InfoTooltip>
      <InfoTooltip content="This is a tooltip on the left" position={{ side: "left" }}>
        Left Tooltip
      </InfoTooltip>
      <InfoTooltip content="This is a tooltip on the top" position={{ side: "top" }}>
        Top Tooltip
      </InfoTooltip>
    </Row>
    <Row>
      <InfoTooltip content="This is a tooltip on the bottom" position={{ side: "bottom" }}>
        Bottom Tooltip
      </InfoTooltip>
      <InfoTooltip
        content="This tooltip has custom alignment"
        position={{
          side: "right",
          align: "start",
          sideOffset: 50,
        }}
      >
        Custom Alignment
      </InfoTooltip>
      <InfoTooltip content="This tooltip is disabled" disabled>
        Disabled Tooltip
      </InfoTooltip>
    </Row>
  </div>`}
      >
        <div className="flex flex-col gap-8 p-4">
          <Row>
            <InfoTooltip content="This is a tooltip on the right">Right Tooltip</InfoTooltip>
            <InfoTooltip content="This is a tooltip on the left" position={{ side: "left" }}>
              Left Tooltip
            </InfoTooltip>
            <InfoTooltip content="This is a tooltip on the top" position={{ side: "top" }}>
              Top Tooltip
            </InfoTooltip>
          </Row>
          <Row>
            <InfoTooltip content="This is a tooltip on the bottom" position={{ side: "bottom" }}>
              Bottom Tooltip
            </InfoTooltip>
            <InfoTooltip
              content="This tooltip has custom alignment"
              position={{
                side: "right",
                align: "start",
                sideOffset: 50,
              }}
            >
              Custom Alignment
            </InfoTooltip>
            <InfoTooltip content="This tooltip is disabled" disabled>
              Disabled Tooltip
            </InfoTooltip>
          </Row>
        </div>
      </RenderComponentWithSnippet>
    </div>
  );
}
