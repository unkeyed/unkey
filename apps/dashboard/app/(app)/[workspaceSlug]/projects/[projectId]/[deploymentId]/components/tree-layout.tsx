import type { Point, TreeLayoutProps, TreeNode } from "./types";

export function TreeLayout<T extends TreeNode>({
  data,
  nodeSpacing = { x: 200, y: 150 },
  direction = "vertical",
  renderNode,
  renderConnection,
}: TreeLayoutProps<T>) {
  // Calculate all node positions
  const positionedNodes = calculateTreeLayout(data, nodeSpacing, direction);

  // Collect all connections
  const connections: Array<{
    from: Point;
    to: Point;
    parent: T;
    child: T;
  }> = [];

  positionedNodes.forEach((positioned) => {
    if (positioned.node.children) {
      positioned.node.children.forEach((child) => {
        const childPositioned = positionedNodes.find(
          (p) => p.node.id === child.id
        );
        if (childPositioned) {
          connections.push({
            from: positioned.position,
            to: childPositioned.position,
            parent: positioned.node,
            child: childPositioned.node as T,
          });
        }
      });
    }
  });

  return (
    <>
      {/* Render connections first (behind nodes) */}
      {connections.map((conn, idx) =>
        renderConnection ? (
          renderConnection(conn.from, conn.to, conn.parent, conn.child)
        ) : (
          <AnimatedConnectionLine
            key={`${conn.parent.id}-${conn.child.id}`}
            id={`${conn.parent.id}-${conn.child.id}-${idx}`}
            from={conn.from}
            to={conn.to}
          />
        )
      )}

      {/* Render nodes on top */}
      {positionedNodes.map((positioned) =>
        renderNode(positioned.node, positioned.position)
      )}
    </>
  );
}

function calculateTreeLayout<T extends TreeNode>(
  root: T,
  spacing: { x: number; y: number },
  direction: "vertical" | "horizontal"
): PositionedNode<T>[] {
  const positioned: PositionedNode<T>[] = [];

  // BFS to assign levels
  const queue: Array<{ node: T; level: number; parent: T | null }> = [
    { node: root, level: 0, parent: null },
  ];

  const levelGroups: Map<number, T[]> = new Map();

  while (queue.length > 0) {
    const { node, level } = queue.shift()!;

    if (!levelGroups.has(level)) {
      levelGroups.set(level, []);
    }
    levelGroups.get(level)!.push(node);

    if (node.children) {
      node.children.forEach((child) => {
        queue.push({ node: child as T, level: level + 1, parent: node });
      });
    }
  }

  // Position nodes by level
  levelGroups.forEach((nodes, level) => {
    nodes.forEach((node, index) => {
      const position =
        direction === "vertical"
          ? {
              x: (index - (nodes.length - 1) / 2) * spacing.x,
              y: level * spacing.y,
            }
          : {
              x: level * spacing.x,
              y: (index - (nodes.length - 1) / 2) * spacing.y,
            };

      positioned.push({ node, position, level });
    });
  });

  return positioned;
}

// AnimatedConnectionLine component
type ConnectionLine = {
  id: string;
  from: Point;
  to: Point;
};

function AnimatedConnectionLine({ from, to, id }: ConnectionLine) {
  return (
    <>
      <defs>
        <linearGradient id={`flow-gradient-${id}`}>
          <stop offset="0%" stopColor="#D8F4F6" stopOpacity="0.02">
            <animate
              attributeName="offset"
              values="-0.5;1"
              dur="2s"
              repeatCount="indefinite"
            />
          </stop>
          <stop offset="10%" stopColor="#FCFDFF" stopOpacity="0.94">
            <animate
              attributeName="offset"
              values="-0.4;1.1"
              dur="2s"
              repeatCount="indefinite"
            />
          </stop>
          <stop offset="30%" stopColor="#FCFDFF" stopOpacity="0.94">
            <animate
              attributeName="offset"
              values="-0.2;1.3"
              dur="2s"
              repeatCount="indefinite"
            />
          </stop>
          <stop offset="40%" stopColor="#D8F4F6" stopOpacity="0.02">
            <animate
              attributeName="offset"
              values="-0.1;1.4"
              dur="2s"
              repeatCount="indefinite"
            />
          </stop>
        </linearGradient>
      </defs>
      <line
        x1={from.x}
        y1={from.y}
        x2={to.x}
        y2={to.y}
        stroke="#212225"
        strokeWidth="2"
      />
      <line
        x1={from.x}
        y1={from.y}
        x2={to.x}
        y2={to.y}
        stroke={`url(#flow-gradient-${id})`}
        strokeWidth="2"
      />
    </>
  );
}
