import type { JSX } from 'solid-js';

interface ButtonProps {
  onClick?: () => void;
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
  size?: 'sm' | 'md';
  disabled?: boolean;
  children: JSX.Element;
  class?: string;
}

const variants = {
  primary: 'bg-blue-600 text-white hover:bg-blue-700',
  secondary: 'bg-gray-200 text-gray-800 hover:bg-gray-300 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600',
  danger: 'bg-red-600 text-white hover:bg-red-700',
  ghost: 'text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800',
};

const sizes = {
  sm: 'px-2 py-1 text-xs',
  md: 'px-3 py-1.5 text-sm',
};

export default function Button(props: ButtonProps) {
  return (
    <button
      onClick={props.onClick}
      disabled={props.disabled}
      class={`rounded font-medium transition-colors disabled:opacity-50 cursor-pointer ${variants[props.variant || 'primary']} ${sizes[props.size || 'md']} ${props.class || ''}`}
    >
      {props.children}
    </button>
  );
}
