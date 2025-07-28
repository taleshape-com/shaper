import React from 'react';
import { Column } from '../lib/dashboard';

export const ChartHoverContext = React.createContext<{
  hoveredIndex: string | number | null;
  hoveredChartId: string | null;
  hoveredIndexType: Column['type'] | null;
  setHoverState: (index: string | number | null, chartId: string | null, indexType: Column['type'] | null) => void;
}>({
  hoveredIndex: null,
  hoveredChartId: null,
  hoveredIndexType: null,
  setHoverState: () => { },
});


