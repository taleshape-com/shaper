import "./index.css";
import { EmbedComponent, EmbedProps } from "./components/Embed";

export function Dashboard(props: EmbedProps) {
  return (
    <div className="shaper-scope">
      <EmbedComponent {...props} />
    </div>
  );
};
