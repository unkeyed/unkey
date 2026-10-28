import {
  IconCodeBranchOutline18,
  IconCubeOutline18,
  IconEarthOutline18,
  IconGridOutline18,
  IconLink4Outline18,
  IconMicrochipOutline18,
  IconShieldKeyOutline18,
} from "@unkey/icons";
import { Card, Skeleton } from "@unkey/ui";
import { cn } from "cn";
import {
  Canvas,
  GhostNode,
  Group,
  HFlow,
  IconRow,
  Node,
  NodeHeader,
  RowItem,
  Rows,
} from "./canvas/nodes";

function Bar({ className }: { className: string }) {
  return <Skeleton className={cn("inline-block h-3 rounded align-middle", className)} />;
}

export function AppCanvasSkeleton() {
  return (
    <Card className="flex flex-col">
      <Canvas label="App canvas loading" focusable={false}>
        <Group icon={<IconEarthOutline18 />} label="Domains">
          <Node>
            <NodeHeader icon={<IconEarthOutline18 />} name="Domains" />
            <Rows>
              <IconRow icon={<IconEarthOutline18 />}>
                <Bar className="w-40" />
              </IconRow>
            </Rows>
          </Node>
          <GhostNode
            icon={<IconLink4Outline18 />}
            name={<Bar className="w-28" />}
            description={<Bar className="w-44" />}
          />
        </Group>
        <HFlow empty />
        <Group icon={<IconCubeOutline18 />} label="App">
          <Node>
            <NodeHeader
              icon={<Skeleton className="size-3 rounded-full" />}
              name={<Bar className="w-16" />}
              right={<Bar className="w-10" />}
            />
            <Rows>
              <RowItem
                icon={<IconCodeBranchOutline18 />}
                label="Branch"
                value={<Bar className="w-20" />}
              />
              <RowItem
                icon={<IconEarthOutline18 />}
                label="Regions"
                value={<Bar className="w-10" />}
              />
              <RowItem
                icon={<IconMicrochipOutline18 />}
                label="Resources"
                value={<Bar className="w-20" />}
              />
            </Rows>
          </Node>
        </Group>
        <HFlow empty />
        <Group icon={<IconGridOutline18 />} label="Services">
          <Node>
            <NodeHeader icon={<IconShieldKeyOutline18 />} name="Policies" />
            <Rows>
              <IconRow icon={<IconShieldKeyOutline18 />}>
                <Bar className="w-3/4" />
              </IconRow>
              <IconRow icon={<IconShieldKeyOutline18 />}>
                <Bar className="w-2/3" />
              </IconRow>
              <IconRow icon={<IconShieldKeyOutline18 />}>
                <Bar className="w-1/2" />
              </IconRow>
            </Rows>
          </Node>
        </Group>
      </Canvas>
    </Card>
  );
}
