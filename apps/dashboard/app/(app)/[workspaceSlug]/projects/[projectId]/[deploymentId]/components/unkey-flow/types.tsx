export type Point = { x: number; y: number };

export type CanvasCustomState = {
  scale: number;
  offset: Point;
};

export type InfiniteCanvasProps = {
  minZoom?: number;
  maxZoom?: number;
  zoomSpeed?: number;
  gridSize?: number;
  gridDotSize?: number;
  gridDotColor?: string;
  showGrid?: boolean;
  onViewChange?: (state: CanvasCustomState) => void;
  children: React.ReactNode;
};

export type TreeNode = {
  id: string;
  direction?: "vertical" | "horizontal";
  children?: TreeNode[];
  size?: { width: number; height: number };
  [key: string]: unknown;
};

export type TreeLayoutProps<T extends TreeNode> = {
  data: T;
  nodeSpacing?: { x: number; y: number };
  onNodeClick?: (node: T) => void;
  renderNode: (node: T, position: Point, parent: T | undefined) => React.ReactNode;
  renderConnection?: (path: Point[], parent: T, child: T) => React.ReactNode;
};

export type PositionedNode<T> = {
  node: T;
  position: Point;
  level: number;
};
