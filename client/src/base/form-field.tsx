import React from "react";
import * as Form from "@radix-ui/react-form";

type FormFieldProps = React.InputHTMLAttributes<HTMLInputElement> & {
  name: string;
  title: string;
  messages: Partial<{ [keys: string]: Array<String> }> | undefined;
};

export const FormField = React.forwardRef<HTMLInputElement, FormFieldProps>(
  ({ title, messages, name, type, ...props }: FormFieldProps, ref: React.Ref<HTMLInputElement>) => {

    return <Form.Field className="grid mb-[10px]" name={name} {...props} ref={ref}>
      <div className="flex items-baseline justify-between">
        <Form.Label className="text-xl font-medium leading-[35px] text-white">
          {title}
        </Form.Label>
        {messages && name in messages
          ? (
            <div className="flex flex-col-reverse text-right">
              {(() => {
                const m = messages[name]
                if (!m) {
                  return null
                }

                return <>{m.map((message, key) =>
                  <Form.Message key={key} className="text-base text-red-400 font-italic opacity-[0.8]">
                    {message}
                  </Form.Message>
                )}</>
              })()}
            </div>
          )
          : null}
      </div>
      <Form.Control asChild>
        <input type={type}
          className="box-border w-full bg-white text-black shadow-blackA9 inline-flex h-[35px] appearance-none items-center justify-center rounded-[4px] px-[10px] text-lg leading-none shadow-[0_0_0_1px] outline-none hover:shadow-[0_0_0_1px_cornflowerblue] focus:shadow-[0_0_0_1px_cornflowerblue] selection:color-white selection:bg-blackA9" />
      </Form.Control>
    </Form.Field>
  },

)

