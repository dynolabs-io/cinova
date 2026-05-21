// https://docs.expo.dev/guides/using-eslint/
// Mirrors dynolabs-io/vcard/eslint.config.js verbatim — flat-config + expo preset.
const { defineConfig } = require('eslint/config');
const expoConfig = require('eslint-config-expo/flat');

module.exports = defineConfig([
  expoConfig,
  {
    ignores: ['dist/*', 'node_modules/*', '.expo/*', 'ios/*', 'android/*'],
  },
]);
