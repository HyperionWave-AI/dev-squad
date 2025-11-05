import React from 'react';
import { Label } from '@atoms/Label';
import { Input, InputProps } from '@atoms/Input';
import { Textarea, TextareaProps } from '@atoms/Textarea';
import { cn } from '@/utils';

export interface FormFieldProps {
  label: string;
  id: string;
  error?: string;
  helperText?: string;
  required?: boolean;
  className?: string;
  type?: 'input' | 'textarea';
  inputProps?: InputProps;
  textareaProps?: TextareaProps;
}

export function FormField({
  label,
  id,
  error,
  helperText,
  required,
  className,
  type = 'input',
  inputProps,
  textareaProps,
}: FormFieldProps) {
  return (
    <div className={cn('space-y-2', className)}>
      <Label htmlFor={id}>
        {label}
        {required && <span className="text-red-500 ml-1">*</span>}
      </Label>
      {type === 'input' ? (
        <Input
          id={id}
          aria-invalid={!!error}
          aria-describedby={error ? `${id}-error` : helperText ? `${id}-helper` : undefined}
          {...inputProps}
        />
      ) : (
        <Textarea
          id={id}
          aria-invalid={!!error}
          aria-describedby={error ? `${id}-error` : helperText ? `${id}-helper` : undefined}
          {...textareaProps}
        />
      )}
      {error && (
        <p id={`${id}-error`} className="text-sm text-red-500">
          {error}
        </p>
      )}
      {helperText && !error && (
        <p id={`${id}-helper`} className="text-sm text-gray-500">
          {helperText}
        </p>
      )}
    </div>
  );
}
