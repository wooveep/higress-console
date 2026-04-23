import { h } from 'vue';
import i18n from '@/i18n';

let antFeedbackModulePromise: Promise<typeof import('ant-design-vue')> | null = null;

function loadAntFeedbackModule() {
  antFeedbackModulePromise ??= import('ant-design-vue');
  return antFeedbackModulePromise;
}

export function showSuccess(content: string) {
  void loadAntFeedbackModule().then(({ message }) => {
    message.success(content);
  });
}

export function showError(content: string) {
  void loadAntFeedbackModule().then(({ message }) => {
    message.error(content);
  });
}

export function showWarning(content: string) {
  void loadAntFeedbackModule().then(({ message }) => {
    message.warning(content);
  });
}

export function showInfo(content: string) {
  void loadAntFeedbackModule().then(({ message }) => {
    message.info(content);
  });
}

export function showConfirm(config: Record<string, any>) {
  void loadAntFeedbackModule().then(({ Modal }) => {
    Modal.confirm(config);
  });
}

export function showWarningModal(config: Record<string, any>) {
  void loadAntFeedbackModule().then(({ Modal }) => {
    Modal.warning(config);
  });
}

export function showCopyValueModal(config: {
  title: string;
  value: string;
  message?: string;
  width?: number;
}) {
  void loadAntFeedbackModule().then(({ Modal, Button, Input, Space }) => {
    const value = config.value || '';
    const copyValue = async () => {
      try {
        await navigator.clipboard.writeText(value);
        showSuccess(i18n.global.t('misc.copySuccess'));
      } catch {
        showError(i18n.global.t('misc.copyError'));
      }
    };

    Modal.info({
      title: config.title,
      width: config.width ?? 560,
      okText: i18n.global.t('misc.close'),
      content: h(
        Space,
        {
          direction: 'vertical',
          size: 12,
          style: {
            width: '100%',
          },
        },
        {
          default: () => [
            config.message
              ? h(
                'div',
                {
                  class: 'portal-copy-modal__message',
                },
                config.message,
              )
              : null,
            h(Input.TextArea, {
              value,
              autoSize: {
                minRows: 2,
                maxRows: 4,
              },
              readonly: true,
            }),
            h(
              Button,
              {
                type: 'primary',
                onClick: copyValue,
              },
              {
                default: () => i18n.global.t('misc.copy'),
              },
            ),
          ],
        },
      ),
    });
  });
}
