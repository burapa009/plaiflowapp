export type BillingInterval = "monthly" | "six_months" | "yearly";

export type PlanDefinition = {
  key: "Free" | "Starter" | "Business" | "Growth" | "AccountingFirm";
  name: string;
  recommended?: boolean;
  prices: Record<BillingInterval, {
    interval: BillingInterval;
    months: number;
    total_satang: number;
    effective_month_satang: number;
    saving_percent: number;
  }>;
  entitlements: Record<string, boolean>;
  limits: { members: number; line_groups: number; documents_per_month: number; client_relationships?: number };
};

export function formatSatang(satang: number) {
  return new Intl.NumberFormat("th-TH", { style: "currency", currency: "THB", minimumFractionDigits: 2 }).format(satang / 100);
}

export function intervalLabel(interval: BillingInterval) {
  return { monthly: "รายเดือน", six_months: "6 เดือน", yearly: "รายปี" }[interval];
}

export function isBillingInterval(value: string | undefined): value is BillingInterval {
  return value === "monthly" || value === "six_months" || value === "yearly";
}
