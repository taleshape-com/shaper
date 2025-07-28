import React, { useEffect, useContext } from "react";
import { RiMenuLine } from "@remixicon/react";
import { MenuContext } from "../contexts/MenuContext";

export function MenuTrigger({
  children,
  className,
  title,
}: {
  children?: React.ReactNode;
  className?: string;
  title?: string;
}) {
  const { isMenuOpen, setIsMenuOpen, setExtraContent, setTitle } = useContext(MenuContext);

  useEffect(() => {
    setExtraContent(children);
  }, [children, setExtraContent]);

  useEffect(() => {
    setTitle(title);
  }, [title, setTitle]);


  if (isMenuOpen) {
    return null;
  }

  return (
    <button
      className={className}
      onClick={() => {
        setIsMenuOpen(true);
      }}
    >
      <RiMenuLine className="py-1 size-7 text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors" />
    </button>
  );
}

