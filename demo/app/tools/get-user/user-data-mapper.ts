import dayjs from "dayjs";
import { v4 as uuidv4 } from "uuid";

type Row = Record<string, unknown>;

export default async function outputTransform(payload: { results?: Row[] }) {
  const rows = payload?.results ?? [];
  console.log("outputTransform", rows);
  return rows.map((row) => ({
    trace_id: uuidv4(),
    id: row.id,
    name: row.name,
    email: row.email,
    created_at_iso: row.created_at ? dayjs(String(row.created_at)).toISOString() : null,
  }));
}
