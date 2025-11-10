import { useCallback, useEffect, useRef, useState } from "react";
import type { Point, PositionedNode, TreeLayoutProps, TreeNode } from "./types";

export function TreeLayout<T extends TreeNode>({
  data,
  nodeSpacing = { x: 50, y: 50 },
  direction = "vertical",
  renderNode,
  renderConnection,
}: TreeLayoutProps<T>) {
  const [nodeSizes, setNodeSizes] = useState<
    Map<string, { width: number; height: number }>
  >(new Map());
  const [layoutReady, setLayoutReady] = useState(false);

  // Flatten tree to get all nodes
  const allNodes = flattenTree(data);

  // Once all nodes measured, mark layout as ready
  useEffect(() => {
    if (nodeSizes.size === allNodes.length && !layoutReady) {
      setLayoutReady(true);
    }
  }, [nodeSizes.size, allNodes.length, layoutReady]);

  const handleNodeMeasure = useCallback(
    (id: string, width: number, height: number) => {
      setNodeSizes((prev) => {
        if (prev.get(id)?.width === width && prev.get(id)?.height === height) {
          return prev; // Avoid unnecessary re-renders
        }
        const next = new Map(prev);
        next.set(id, { width, height });
        return next;
      });
    },
    []
  );

  // Calculate layout - dynamic if ready, otherwise rough initial layout
  const positionedNodes = layoutReady
    ? calculateDynamicTreeLayout(data, nodeSpacing, direction, nodeSizes)
    : calculateTreeLayout(data, nodeSpacing, direction);

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
      {/* Render connections first (behind nodes) - only when layout ready */}
      {layoutReady &&
        connections.map((conn, idx) =>
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
      {positionedNodes.map((positioned) => (
        <MeasuredNodeWrapper
          key={positioned.node.id}
          id={positioned.node.id}
          x={positioned.position.x}
          y={positioned.position.y}
          onMeasure={handleNodeMeasure}
        >
          {renderNode(positioned.node, positioned.position)}
        </MeasuredNodeWrapper>
      ))}
    </>
  );
}

// Helper to flatten tree
function flattenTree<T extends TreeNode>(root: T): T[] {
  const result: T[] = [root];
  if (root.children) {
    root.children.forEach((child) => {
      result.push(...flattenTree(child as T));
    });
  }
  return result;
}

// Wrapper that measures node size
type MeasuredNodeWrapperProps = {
  id: string;
  x: number;
  y: number;
  children: React.ReactNode;
  onMeasure: (id: string, width: number, height: number) => void;
};

function MeasuredNodeWrapper({
  id,
  x,
  y,
  children,
  onMeasure,
}: MeasuredNodeWrapperProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (ref.current) {
      const { width, height } = ref.current.getBoundingClientRect();
      onMeasure(id, width, height);
    }
  }, [id, onMeasure]);

  return (
    <foreignObject x={x} y={y} width={1} height={1} overflow="visible">
      <div
        ref={ref}
        style={{
          position: "absolute",
          left: 0,
          top: 0,
          transform: "translate(-50%, -50%)",
        }}
      >
        {children}
      </div>
    </foreignObject>
  );
}

// Initial rough layout (before measurement)
function calculateTreeLayout<T extends TreeNode>(
  root: T,
  spacing: { x: number; y: number },
  direction: "vertical" | "horizontal"
): PositionedNode<T>[] {
  const positioned: PositionedNode<T>[] = [];
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

  levelGroups.forEach((nodes, level) => {
    nodes.forEach((node, index) => {
      const position =
        direction === "vertical"
          ? {
              x: (index - (nodes.length - 1) / 2) * 200, // Rough spacing
              y: level * 150,
            }
          : {
              x: level * 200,
              y: (index - (nodes.length - 1) / 2) * 150,
            };

      positioned.push({ node, position, level });
    });
  });

  return positioned;
}

// Dynamic layout with actual node sizes
// Dynamic layout with actual node sizes
function calculateDynamicTreeLayout<T extends TreeNode>(
  root: T,
  spacing: { x: number; y: number },
  direction: "vertical" | "horizontal",
  nodeSizes: Map<string, { width: number; height: number }>
): PositionedNode<T>[] {
  const positioned: PositionedNode<T>[] = [];
  const queue: Array<{ node: T; level: number }> = [{ node: root, level: 0 }];
  const levelGroups: Map<number, T[]> = new Map();

  while (queue.length > 0) {
    const { node, level } = queue.shift()!;

    if (!levelGroups.has(level)) {
      levelGroups.set(level, []);
    }
    levelGroups.get(level)!.push(node);

    if (node.children) {
      node.children.forEach((child) => {
        queue.push({ node: child as T, level: level + 1 });
      });
    }
  }

  if (direction === "vertical") {
    // Calculate Y positions based on max height per level
    const levelYPositions: number[] = [];
    let currentY = 0;

    levelGroups.forEach((nodes, level) => {
      const maxHeightInLevel = Math.max(
        ...nodes.map((n) => {
          const size = nodeSizes.get(n.id);
          console.log(`Node ${n.id} size:`, size); // DEBUG
          return size?.height || 100; // Fallback to 100 if not measured yet
        })
      );
      levelYPositions[level] = currentY;
      currentY += maxHeightInLevel + spacing.y;
    });

    // Calculate X positions based on actual widths
    levelGroups.forEach((nodes, level) => {
      let currentX = 0;

      // First calculate total width to center
      const totalWidth = nodes.reduce((sum, n) => {
        const size = nodeSizes.get(n.id) || { width: 100, height: 100 }; // Fallback
        return sum + size.width + spacing.x;
      }, -spacing.x);

      currentX = -totalWidth / 2;

      nodes.forEach((node) => {
        const size = nodeSizes.get(node.id) || { width: 100, height: 100 };
        console.log(
          `Positioning node ${node.id} at x=${
            currentX + size.width / 2
          }, width=${size.width}`
        ); // DEBUG

        const position = {
          x: currentX + size.width / 2,
          y: levelYPositions[level],
        };

        positioned.push({ node, position, level });
        currentX += size.width + spacing.x;
      });
    });
  } else {
    // Horizontal layout
    const levelXPositions: number[] = [];
    let currentX = 0;

    levelGroups.forEach((nodes, level) => {
      const maxWidthInLevel = Math.max(
        ...nodes.map((n) => nodeSizes.get(n.id)?.width || 100)
      );
      levelXPositions[level] = currentX;
      currentX += maxWidthInLevel + spacing.x;
    });

    levelGroups.forEach((nodes, level) => {
      const totalHeight = nodes.reduce(
        (sum, n) => sum + (nodeSizes.get(n.id)?.height || 100) + spacing.y,
        -spacing.y
      );
      let currentY = -totalHeight / 2;

      nodes.forEach((node) => {
        const size = nodeSizes.get(node.id) || { width: 100, height: 100 };
        const position = {
          x: levelXPositions[level],
          y: currentY + size.height / 2,
        };

        positioned.push({ node, position, level });
        currentY += size.height + spacing.y;
      });
    });
  }

  return positioned;
}

// AnimatedConnectionLine component

type ConnectionLine = {
  id: string;
  from: Point;
  to: Point;
};

export function AnimatedConnectionLine({ from, to, id }: ConnectionLine) {
  const pathRef = useRef<SVGPathElement>(null);
  const [pathLength, setPathLength] = useState(0);

  useEffect(() => {
    if (pathRef.current) {
      setPathLength(pathRef.current.getTotalLength());
    }
  }, [from, to]);

  const isVertical = Math.abs(from.x - to.x) < 10;
  let pathD: string;

  if (isVertical) {
    pathD = `M ${from.x} ${from.y} L ${to.x} ${to.y}`;
  } else {
    const radius = 32;
    const direction = to.x > from.x ? 1 : -1;
    const verticalHeight = 36;
    const y1 = from.y + verticalHeight;
    const corner1CenterY = y1 + radius;
    const corner1X = from.x + radius * direction;
    const horizontalEndX = to.x - radius * direction;
    const corner2Y = to.y - (32 + radius);
    pathD = `
      M ${from.x} ${from.y}
      L ${from.x} ${y1}
      Q ${from.x} ${corner1CenterY} ${corner1X} ${corner1CenterY}
      L ${horizontalEndX} ${corner1CenterY}
      Q ${to.x} ${corner1CenterY} ${to.x} ${corner2Y}
      L ${to.x} ${to.y}
    `
      .replace(/\s+/g, " ")
      .trim();
  }

  const lightBandSize = 40; // pixels
  const gapSize = pathLength - lightBandSize;
  const dashArray = `${lightBandSize} ${gapSize}`;

  // Constant velocity
  const velocity = 100; // px/s
  const duration = pathLength / velocity;

  return (
    <>
      {/* Base line - dark */}
      <path
        ref={pathRef}
        d={pathD}
        className="stroke-gray-3"
        strokeWidth="2"
        fill="none"
        strokeLinecap="round"
      />

      {/* Animated light band */}
      <path
        d={pathD}
        className="stroke-grayA-12"
        strokeWidth="2"
        fill="none"
        strokeLinecap="round"
        strokeDasharray={dashArray}
        strokeDashoffset={pathLength}
        style={{
          opacity: 0.94,
        }}
      >
        <animate
          attributeName="stroke-dashoffset"
          from={pathLength}
          to={0}
          dur={`${duration}s`}
          repeatCount="indefinite"
        />
      </path>
    </>
  );
}
