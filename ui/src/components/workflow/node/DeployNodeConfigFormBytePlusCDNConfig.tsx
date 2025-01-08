import { useTranslation } from "react-i18next";
import { Form, type FormInstance, Input } from "antd";
import { createSchemaFieldRule } from "antd-zod";
import { z } from "zod";

import { validDomainName } from "@/utils/validators";

type DeployNodeConfigFormBytePlusCDNConfigFieldValues = Nullish<{
  domain: string;
}>;

export type DeployNodeConfigFormBytePlusCDNConfigProps = {
  form: FormInstance;
  formName: string;
  disabled?: boolean;
  initialValues?: DeployNodeConfigFormBytePlusCDNConfigFieldValues;
  onValuesChange?: (values: DeployNodeConfigFormBytePlusCDNConfigFieldValues) => void;
};

const initFormModel = (): DeployNodeConfigFormBytePlusCDNConfigFieldValues => {
  return {};
};

const DeployNodeConfigFormBytePlusCDNConfig = ({
  form: formInst,
  formName,
  disabled,
  initialValues,
  onValuesChange,
}: DeployNodeConfigFormBytePlusCDNConfigProps) => {
  const { t } = useTranslation();

  const formSchema = z.object({
    domain: z
      .string({ message: t("workflow_node.deploy.form.byteplus_cdn_domain.placeholder") })
      .refine((v) => validDomainName(v, true), t("common.errmsg.domain_invalid")),
  });
  const formRule = createSchemaFieldRule(formSchema);

  const handleFormChange = (_: unknown, values: z.infer<typeof formSchema>) => {
    onValuesChange?.(values);
  };

  return (
    <Form
      form={formInst}
      disabled={disabled}
      initialValues={initialValues ?? initFormModel()}
      layout="vertical"
      name={formName}
      onValuesChange={handleFormChange}
    >
      <Form.Item
        name="domain"
        label={t("workflow_node.deploy.form.byteplus_cdn_domain.label")}
        rules={[formRule]}
        tooltip={<span dangerouslySetInnerHTML={{ __html: t("workflow_node.deploy.form.byteplus_cdn_domain.tooltip") }}></span>}
      >
        <Input placeholder={t("workflow_node.deploy.form.byteplus_cdn_domain.placeholder")} />
      </Form.Item>
    </Form>
  );
};

export default DeployNodeConfigFormBytePlusCDNConfig;
