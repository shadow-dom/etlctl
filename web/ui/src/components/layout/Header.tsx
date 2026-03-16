import type { JSX } from 'solid-js';

interface HeaderProps {
  title: string;
  actions?: JSX.Element;
}

export default function Header(props: HeaderProps) {
  return (
    <header class="h-14 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between px-6 bg-white dark:bg-gray-800">
      <h1 class="text-lg font-semibold text-gray-900 dark:text-white m-0">{props.title}</h1>
      <div class="flex items-center gap-2">
        {props.actions}
      </div>
    </header>
  );
}
