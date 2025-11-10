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
  children?: TreeNode[];
  [key: string]: unknown;
};

export type TreeLayoutProps<T extends TreeNode> = {
  data: T;
  nodeSpacing?: { x: number; y: number };
  siblingSpacing?: { x: number; y: number };
  direction?: "vertical" | "horizontal";
  renderNode: (node: T, position: Point) => React.ReactNode;
  renderConnection?: (
    from: Point,
    to: Point,
    parentNode: T,
    childNode: T
  ) => React.ReactNode;
};

export type PositionedNode<T> = {
  node: T;
  position: Point;
  level: number;
};
