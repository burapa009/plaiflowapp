const priorities = new Set(["Normal", "High", "Urgent"]);
const statuses = new Set(["Open", "InProgress", "Done", "Cancelled"]);

export function taskFormBody(formData: FormData) {
  const title = String(formData.get("title") ?? "").trim();
  const description = String(formData.get("description") ?? "");
  const priority = String(formData.get("priority") ?? "Normal");
  const dueOn = String(formData.get("due_on") ?? "");
  const assignee = String(formData.get("assignee_user_id") ?? "");
  if (Array.from(title).length < 1 || Array.from(title).length > 200 || Array.from(description).length > 5000 ||
    !priorities.has(priority) || !validDate(dueOn)) return null;
  return new URLSearchParams({ title, description, priority, due_on: dueOn, assignee_user_id: assignee });
}

export function updateTaskBody(formData: FormData) {
  const body = taskFormBody(formData);
  const status = String(formData.get("status") ?? "");
  if (!body || !statuses.has(status)) return null;
  body.set("status", status);
  return body;
}

function validDate(value: string) {
  if (!value) return true;
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  const parsed = new Date(`${value}T00:00:00Z`);
  return !Number.isNaN(parsed.valueOf()) && parsed.toISOString().startsWith(value);
}
