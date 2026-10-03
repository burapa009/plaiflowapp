export function formatReviewAmount(value: string, currency: string) {
  if (!value) return "—";
  if (!/^-?(?:\d+|\d{1,3}(?:,\d{3})+)(?:\.\d{1,2})?$/.test(value)) return value;
  const amount = value.includes(",") ? value : value.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return `${currency === "THB" ? "฿" : currency ? `${currency} ` : ""}${amount}`;
}
