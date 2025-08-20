// SPDX-License-Identifier: MPL-2.0

import React from "react";

export function PreviewError({ children }: { children: React.ReactNode }) {
  return (
    <div className="fixed w-full h-full p-4 z-50 backdrop-blur-sm flex justify-center">
      <div className="p-4 bg-red-100 text-red-700 rounded mt-32 h-fit">
        {children}
      </div>
    </div>
  );
}


