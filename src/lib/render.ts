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
    return d.toISOString().substring(0, 7)
  }
  if (columnType === "date") {
    return d.toISOString().substring(0, 10)
  }
  if (columnType === "hour") {
    return d.toISOString().substring(0, 16).replace('T', ' ')
  }
  if (columnType === "timestamp") {
    return d.toISOString().substring(0, 19).replace('T', ' ')
  }
  return value;
};

