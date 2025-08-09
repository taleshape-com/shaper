// SPDX-License-Identifier: MPL-2.0

import React from 'react';
import { Column } from '../lib/types';

// Coordinate to display cursor across charts when hovering
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


