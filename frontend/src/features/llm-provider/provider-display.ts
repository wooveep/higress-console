import type { LlmProvider } from '@/interfaces/llm-provider';
import { getProviderTypeLabel } from './provider-form';

type Translate = (key: string, params?: Record<string, unknown>) => string;

export function formatProviderDisplayName(
  providerName: string,
  providers: Array<LlmProvider | null | undefined>,
  t: Translate,
) {
  const normalizedName = String(providerName || '').trim();
  if (!normalizedName) {
    return '';
  }
  const matched = (providers || []).find((item) => item?.name === normalizedName);
  if (!matched) {
    return normalizedName;
  }
  return `${getProviderTypeLabel(matched.type, t)} / ${matched.name}`;
}

export function buildProviderDisplayOptions(
  providers: LlmProvider[],
  t: Translate,
  extraValues: string[] = [],
) {
  const options = [...(providers || [])]
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((item) => ({
      label: formatProviderDisplayName(item.name, providers, t),
      value: item.name,
    }));

  extraValues.forEach((value) => {
    const normalized = String(value || '').trim();
    if (normalized && !options.some((item) => item.value === normalized)) {
      options.unshift({ label: `${normalized}（历史值）`, value: normalized });
    }
  });
  return options;
}
