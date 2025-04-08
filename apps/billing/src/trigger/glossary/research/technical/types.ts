export type ExaCosts = {
  costDollars: {
    total: number;
    search?: {
      neural?: number;
      keyword?: number;
    };
    contents?: {
      text?: number;
      summary?: number;
    };
  };
};
