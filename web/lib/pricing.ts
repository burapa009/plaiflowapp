export type BillingInterval = "monthly" | "six_months" | "yearly";

export type PlanDefinition = {
  key: "Free" | "Starter" | "Business";
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
