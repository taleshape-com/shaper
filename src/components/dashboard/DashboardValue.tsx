import { Column, Result } from "../../lib/dashboard";
import { formatValue } from "../../lib/render";
import { Card } from "../tremor/Card";
import { cx } from "../../lib/utils";

type ValueProps = {
  label?: string;
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
  sectionCount: number;
};

function DashboardValue({ label, headers, data, sectionCount }: ValueProps) {
  const header = headers[0]
  const headerType = header.type
  const value = data[0][0]
  return (
    <div className="p-2 mb-3">
      {label &&
        <h2 className="text-md mb-2 text-center text-slate-700">
          {label}
        </h2>
      }
      <Card className={cx(
        "overflow-auto px-2 py-16 flex items-center justify-center", {
        "min-h-[250px]": label,
        "min-h-[calc(250px+2.00rem)]": !label,
        ["sm:h-[calc(36vh)] lg:h-[calc(60vh-4.1rem)] xl:h-[calc(100vh-8.25rem)]"]: label && sectionCount === 2,
        ["sm:h-[calc(36vh)] lg:h-[calc(60vh-2.1rem)] xl:h-[calc(100vh-6.25rem)]"]: !label && sectionCount === 2,
        ["md:h-[calc(39vh)] 2xl:h-[calc(50vh)]"]: label && sectionCount === 3,
        ["md:h-[calc(43vh)] 2xl:h-[calc(54vh)]"]: !label && sectionCount === 3,
        ["md:h-[calc(50vh-5.7rem)]"]: label && sectionCount >= 4,
        ["md:h-[calc(50vh-3.65rem)]"]: !label && sectionCount >= 4,
      })}>
        <div className="text-center">
          <div className={"text-6xl text-slate-800"}>
            {formatValue(value, headerType)}
          </div>
          <div className="text-lg text-slate-800">
            {header.name}
          </div>
        </div>
      </Card>
    </div>
  );
}

export default DashboardValue;

