export const formatValue = (value: string | number | boolean) => {
  if (typeof value === "number") {
    return value;
  }
  if (typeof value === "boolean") {
    return value ? "YES" : "NO";
  }
  return value;
};

export const toYear = (value: string | number | boolean) => {
  if (typeof value === "boolean") {
    return value ? "YES" : "NO";
  }
  return new Date(value).getFullYear().toString();
};

