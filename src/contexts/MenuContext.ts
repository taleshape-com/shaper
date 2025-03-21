import React from 'react';

export const MenuContext = React.createContext<{
  isMenuOpen: boolean;
  setIsMenuOpen: React.Dispatch<React.SetStateAction<boolean | null>>;
  setExtraContent: React.Dispatch<React.SetStateAction<React.ReactNode>>;
}>({
  isMenuOpen: false,
  setIsMenuOpen: () => { },
  setExtraContent: () => { },
});



