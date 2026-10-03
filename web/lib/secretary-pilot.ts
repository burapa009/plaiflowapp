export function secretaryPilotEnabled(organization: string): boolean {
  return process.env.NEXT_PUBLIC_SECRETARY_ENABLED === "true" &&
    (process.env.SECRETARY_PILOT_ORGANIZATION_IDS ?? "").split(",").some((id) => id.trim() === organization);
}
