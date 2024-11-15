import { Column } from "./dashboard";

export const formatValue = (value: string | number | boolean, columnType: Column['type']) => {
  if (typeof value === "number") {
    return value;
  }
  if (typeof value === "boolean") {
    return value ? "YES" : "NO";
  }
  const d = new Date(value)
  if (columnType === "year") {
    return d.getFullYear().toString();
  }
  if (columnType === "month") {
    return d.toLocaleString(navigator.languages, { month: 'short' });
  }
  if (columnType === "date") {
    return d.toLocaleDateString();
  }
  if (columnType === "hour") {
    return d.toLocaleString(navigator.languages, { year: 'numeric', month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  }
  if (columnType === "timestamp") {
    return d.toLocaleString();
  }
  return value;
};

