import { translations } from "@/lib/translations";

export function useTranslation() {
  // Currently we only support Russian, but this hook makes it easy to add more languages later
  const t = translations;

  return { t };
}
