import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useControllableValue } from "ahooks";
import { Modal, notification } from "antd";

import { type AccessModel } from "@/domain/access";
import { accessProvidersMap } from "@/domain/provider";
import { useTriggerElement, useZustandShallowSelector } from "@/hooks";
import { useAccessesStore } from "@/stores/access";
import { getErrMsg } from "@/utils/error";

import AccessForm, { type AccessFormInstance, type AccessFormProps } from "./AccessForm";

export type AccessEditModalProps = {
  data?: AccessFormProps["initialValues"];
  loading?: boolean;
  open?: boolean;
  preset: AccessFormProps["preset"];
  trigger?: React.ReactNode;
  onOpenChange?: (open: boolean) => void;
  afterSubmit?: (record: AccessModel) => void;
};

const AccessEditModal = ({ data, loading, trigger, preset, afterSubmit, ...props }: AccessEditModalProps) => {
  const { t } = useTranslation();

  const [notificationApi, NotificationContextHolder] = notification.useNotification();

  const { createAccess, updateAccess } = useAccessesStore(useZustandShallowSelector(["createAccess", "updateAccess"]));

  const [open, setOpen] = useControllableValue<boolean>(props, {
    valuePropName: "open",
    defaultValuePropName: "defaultOpen",
    trigger: "onOpenChange",
  });

  const triggerEl = useTriggerElement(trigger, { onClick: () => setOpen(true) });

  const formRef = useRef<AccessFormInstance>(null);
  const [formPending, setFormPending] = useState(false);

  const handleOkClick = async () => {
    setFormPending(true);
    try {
      await formRef.current!.validateFields();
    } catch (err) {
      setFormPending(false);
      throw err;
    }

    try {
      let values: AccessModel = formRef.current!.getFieldsValue();
      values.usage = accessProvidersMap.get(values.provider)!.usage;

      if (preset === "add") {
        if (data?.id) {
          throw "Invalid props: `data`";
        }

        values = await createAccess(values);
      } else if (preset === "edit") {
        if (!data?.id) {
          throw "Invalid props: `data`";
        }

        values = await updateAccess({ ...data, ...values });
      } else {
        throw "Invalid props: `preset`";
      }

      afterSubmit?.(values);
      setOpen(false);
    } catch (err) {
      notificationApi.error({ message: t("common.text.request_error"), description: getErrMsg(err) });

      throw err;
    } finally {
      setFormPending(false);
    }
  };

  const handleCancelClick = () => {
    if (formPending) return;

    setOpen(false);
  };

  return (
    <>
      {NotificationContextHolder}

      {triggerEl}

      <Modal
        afterClose={() => setOpen(false)}
        cancelButtonProps={{ disabled: formPending }}
        closable
        confirmLoading={formPending}
        destroyOnClose
        loading={loading}
        okText={preset === "edit" ? t("common.button.save") : t("common.button.submit")}
        open={open}
        title={t(`access.action.${preset}`)}
        width={480}
        onOk={handleOkClick}
        onCancel={handleCancelClick}
      >
        <div className="pb-2 pt-4">
          <AccessForm ref={formRef} initialValues={data} preset={preset === "add" ? "add" : "edit"} />
        </div>
      </Modal>
    </>
  );
};

export default AccessEditModal;
