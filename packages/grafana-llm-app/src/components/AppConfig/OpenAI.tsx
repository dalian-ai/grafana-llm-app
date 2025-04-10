import React, { ChangeEvent } from 'react';

import { openai } from '@grafana/llm';
import { Field, FieldSet, Input, SecretInput, Select, useStyles2 } from '@grafana/ui';

import { SelectableValue } from '@grafana/data';
import { testIds } from 'components/testIds';
import { getStyles, ProviderType, Secrets, SecretsSet } from './AppConfig';
import { AzureModelDeploymentConfig, AzureModelDeployments } from './AzureConfig';

const OPENAI_API_URL = 'https://api.openai.com';
const AZURE_OPENAI_URL_TEMPLATE = 'https://<resource-name>.openai.azure.com';

export interface OpenAISettings {
  // The URL to reach OpenAI.
  url?: string;
  // The organization ID for OpenAI.
  organizationId?: string;
  // Whether to use Azure OpenAI.
  provider?: ProviderType;
  // A mapping of OpenAI models to Azure deployment names.
  azureModelMapping?: AzureModelDeployments;
  // If the LLM features have been explicitly disabled.
  disabled?: boolean;
}

export function OpenAIConfig({
  settings,
  secrets,
  secretsSet,
  onChange,
  onChangeSecrets,
}: {
  settings: OpenAISettings;
  onChange: (settings: OpenAISettings) => void;
  secrets: Secrets;
  secretsSet: SecretsSet;
  onChangeSecrets: (secrets: Secrets) => void;
}) {
  const s = useStyles2(getStyles);
  // Helper to update settings using the name of the HTML event.
  const onChangeField = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({
      ...settings,
      [event.currentTarget.name]:
        event.currentTarget.type === 'checkbox' ? event.currentTarget.checked : event.currentTarget.value.trim(),
    });
  };

  // Update settings when provider changes, set default URL for OpenAI
  const onChangeProvider = (value: ProviderType) => {
    onChange({
      ...settings,
      provider: value,
      url: value === 'openai' ? OPENAI_API_URL : '',
    });
  };

  return (
    <FieldSet>
      {settings.provider !== 'custom' && 
        <Field label="Provider">
        <Select
          data-testid={testIds.appConfig.provider}
          options={
            [
              { label: 'OpenAI', value: 'openai' },
              { label: 'Azure OpenAI', value: 'azure' },
            ] as Array<SelectableValue<ProviderType>>
          }
          value={settings.provider ?? 'openai'}
          onChange={(e) => onChangeProvider(e.value as ProviderType)}
          width={60}
        />
      </Field>
    }

      <Field
        label={settings.provider === 'azure' ? 'Azure OpenAI Language API Endpoint' : 'API URL'}
        className={s.marginTop}
      >
        <Input
          width={60}
          name="url"
          data-testid={testIds.appConfig.openAIUrl}
          value={settings.provider === 'openai' ? OPENAI_API_URL : settings.url}
          placeholder={
            settings.provider === 'azure' 
              ? AZURE_OPENAI_URL_TEMPLATE
              : settings.provider === 'openai'
                ? OPENAI_API_URL
                : `https://llm.domain.com`
          }
          onChange={onChangeField}
          disabled={settings.provider === 'openai'}
        />
      </Field>

      <Field
        label={settings.provider === 'azure' ? 'Azure OpenAI Key' : 'API Key'}
      >
        <SecretInput
          width={60}
          data-testid={testIds.appConfig.openAIKey}
          name="openAIKey"
          value={secrets.openAIKey}
          isConfigured={secretsSet.openAIKey ?? false}
          placeholder={settings.provider === 'azure' ? '' : 'sk-...'}
          onChange={(e) => onChangeSecrets({ ...secrets, openAIKey: e.currentTarget.value })}
          onReset={() => onChangeSecrets({ ...secrets, openAIKey: '' })}
        />
      </Field>

      {settings.provider === 'openai' && (
        <Field label="API Organization ID">
          <Input
            width={60}
            name="organizationId"
            data-testid={testIds.appConfig.openAIOrganizationID}
            value={settings.organizationId}
            placeholder={settings.organizationId ? '' : 'org-...'}
            onChange={onChangeField}
          />
        </Field>
      )}

      {settings.provider === 'azure' && (
        <Field
          label="Azure OpenAI Model Mapping"
          description="Mapping from model name to Azure deployment name."
        >
          <AzureModelDeploymentConfig
            modelMapping={settings.azureModelMapping ?? []}
            modelNames={Object.values(openai.Model)}
            onChange={(azureModelMapping) =>
              onChange({
                ...settings,
                azureModelMapping,
              })
            }
          />
        </Field>
      )}
    </FieldSet>
  );
}
