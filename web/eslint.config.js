import antfu from '@antfu/eslint-config'

export default antfu(
  {
    vue: true,
    unocss: true,
    markdown: false,
    ignores: [
      '**/public',
      '**/dist*',
    ],
  },
  {
    rules: {
      'e18e/prefer-static-regex': 'off',
      'eslint-comments/no-unlimited-disable': 'off',
      'curly': ['error', 'all'],
      'ts/no-unused-expressions': ['error', {
        allowShortCircuit: true,
        allowTernary: true,
      }],
    },
  },
  {
    files: [
      'src/**/*.vue',
    ],
    rules: {
      'vue/block-order': ['error', {
        order: ['script', 'template', 'style'],
      }],
      // 禁止原生 <select>：展开层是浏览器 UI 不可主题化，一律用 YdSelect（docs/dev/conventions.md §4）
      'vue/no-restricted-html-elements': ['error', {
        elements: ['select'],
        message: '禁止原生 <select>，使用 YdSelect（FaDropdown 封装）替代',
      }],
    },
  },
  {
    files: [
      'pnpm-workspace.yaml',
    ],
    rules: {
      'pnpm/yaml-enforce-settings': 'off',
    },
  },
)
