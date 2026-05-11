// Shared ESLint flat config. Consumed via `import baseConfig from
// "@base/eslint-config";` in per-app eslint.config.mjs.
export default [
  {
    ignores: ["dist/**", "node_modules/**", "**/gen/**"],
  },
  {
    files: ["**/*.{ts,tsx,js,jsx}"],
    rules: {
      "no-console": ["warn", { allow: ["warn", "error"] }],
    },
  },
];
