import dayjs from "dayjs";

type Inputs = {
  city?: string;
  units?: "metric" | "imperial";
};

export async function handler(payload: { inputs?: Inputs; route?: string }) {
  const city = payload?.inputs?.city ?? "Bengaluru";
  const units = payload?.inputs?.units ?? "metric";

  const sampleTemp = units === "imperial" ? 77 : 25;
  const sampleWind = units === "imperial" ? "8 mph" : "13 km/h";

  return [
    {
      city,
      units,
      observed_at: dayjs().toISOString(),
      weather: "Partly cloudy",
      temperature: sampleTemp,
      wind: sampleWind,
      source: "demo-handler",
      route: payload?.route ?? "unknown"
    }
  ];
}
