"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { Row } from "@/app/components/row";
import { Id } from "@unkey/ui";

export const ValueTruncateExample: React.FC = () => (
    <RenderComponentWithSnippet>
        <Row>
            <Id value={"no-truncate-value-used"} />
            <Id value={"6-truncate-value-used"} truncate={6} />
            <Id value={"12-truncate-value-used"} truncate={12} />
        </Row>
    </RenderComponentWithSnippet>
);
