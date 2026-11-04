import {
  IconClockOutline18,
  IconCodeBranchOutline18,
  IconCubeOutline18,
  IconEarthOutline18,
  IconGridOutline18,
  IconMicrochipOutline18,
} from "@unkey/icons";
import { Card, Skeleton } from "@unkey/ui";
import { cn } from "cn";
import {
  Canvas,
  Group,
  HFlow,
  IconRow,
  Node,
  NodeHeader,
  NodeSkeleton,
  RowItem,
  Rows,
} from "./canvas/nodes";

function Bar({ className }: { className: string }) {
  return <Skeleton className={cn("inline-block h-3 rounded align-middle", className)} />;
}

export function AppCanvasSkeleton() {
  return (
    <Card className="flex flex-col" aria-busy="true">
      <output className="sr-only">Loading app...</output>
      <div aria-hidden="true">
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
                  icon={<IconClockOutline18 />}
                  label="Deployed"
                  value={<Bar className="w-16" />}
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
            <NodeSkeleton label="Loading policies" />
          </Group>
        </Canvas>
      </div>
    </Card>
  );
}
