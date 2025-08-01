// SPDX-License-Identifier: MPL-2.0

import React from 'react';

export const MenuContext = React.createContext<{
  isMenuOpen: boolean;
  setIsMenuOpen: React.Dispatch<React.SetStateAction<boolean | null>>;
  setExtraContent: React.Dispatch<React.SetStateAction<React.ReactNode>>;
  setTitle: React.Dispatch<React.SetStateAction<string | undefined>>;
}>({
  isMenuOpen: false,
  setIsMenuOpen: () => { },
  setExtraContent: () => { },
  setTitle: () => { },
});



