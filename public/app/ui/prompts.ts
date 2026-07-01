import type { PromptItem, PromptsResponse } from "./types";

export const PROMPT_IDS = {
  debugSystem: "frontend.debug.system",
  imageDefault: "frontend.image.default",
  videoDefault: "frontend.video.default",
} as const;

const FALLBACKS: Record<string, string> = {
  [PROMPT_IDS.debugSystem]: "Bạn là trợ lý gỡ lỗi hệ thống, hãy trả lời trực tiếp và ngắn gọn.",
  [PROMPT_IDS.imageDefault]: "Một poster sản phẩm sạch sẽ, logo Qwen2API chất liệu kính đặt giữa mặt bàn, ánh sáng studio mềm mại, chi tiết sắc nét",
  [PROMPT_IDS.videoDefault]: "Logo Qwen2API phát sáng từ từ nổi lên từ bàn làm việc tối, ống kính đẩy nhẹ, cảm giác công nghệ, chuyển động mượt mà",
};


export function promptValue(prompts: PromptsResponse | null | undefined, id: string) {
  return normalizePromptsResponse(prompts)?.data.find((item) => item.id === id)?.value ?? FALLBACKS[id] ?? "";
}

export function normalizePromptsResponse(value: unknown): PromptsResponse | null {
  if (!value || typeof value !== "object") {
    return null;
  }

  const payload = value as Partial<PromptsResponse>;
  if (!Array.isArray(payload.data)) {
    return null;
  }

  const rawItems: unknown[] = Array.isArray(payload.data) ? payload.data : [];
  const data = rawItems
    .filter((item): item is Partial<PromptItem> => Boolean(item) && typeof item === "object")
    .map((item) => ({
      id: typeof item.id === "string" ? item.id : "",
      category: typeof item.category === "string" ? item.category : "Chưa phân loại",
      title: typeof item.title === "string" ? item.title : "Prompt chưa đặt tên",
      description: typeof item.description === "string" ? item.description : "",
      defaultValue: typeof item.defaultValue === "string" ? item.defaultValue : "",
      value: typeof item.value === "string" ? item.value : "",
      risk: typeof item.risk === "string" ? item.risk : "",
      placeholders: Array.isArray(item.placeholders)
        ? item.placeholders.filter((placeholder): placeholder is string => typeof placeholder === "string")
        : [],
      modified: Boolean(item.modified),
    }))
    .filter((item) => item.id);

  const categories = Array.isArray(payload.categories)
    ? payload.categories.filter((item): item is string => typeof item === "string")
    : Array.from(new Set(data.map((item) => item.category)));

  return { data, categories };
}
