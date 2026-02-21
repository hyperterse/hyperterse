export default async function inputTransform(payload: { inputs?: Record<string, unknown> }) {
  console.log("inputTransform,,,,,", payload);
  const inputs = payload?.inputs ?? {};
  const rawUserId = inputs.userId;

  if (rawUserId === undefined || rawUserId === null) {
    throw new Error("user_id is required");
  }

  const userId = Number(rawUserId);
  if (!Number.isFinite(userId) || userId <= 0 || !Number.isInteger(userId)) {
    throw new Error("user_id must be a positive integer");
  }

  return {
    ...inputs,
    userId,
  };
}
