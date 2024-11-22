import React from 'react';

export const ChartHoverContext = React.createContext<{
  hoveredIndex: string | number | null;
  hoveredChartId: string | null;
  setHoverState: (index: string | number | null, chartId: string | null) => void;
}>({
  hoveredIndex: null,
  hoveredChartId: null,
  setHoverState: () => { },
});


