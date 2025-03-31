export const LoadingIndicator = () => (
  <div className="fixed bottom-0 left-0 right-0 py-2 bg-background/90 backdrop-blur-sm border-t border-border">
    <div className="flex items-center justify-center gap-2 text-sm text-accent-11">
      <div className="h-1.5 w-1.5 rounded-full bg-accent-11 animate-pulse" />
      Loading more data
    </div>
  </div>
);
