const { getDefaultConfig } = require('expo/metro-config');

const config = getDefaultConfig(__dirname);

// Fix: Zustand (and other packages) ship ESM files (.mjs) that use
// import.meta.env, which is not valid in Metro's non-module web bundle.
// Setting explicit condition names forces Metro to use the 'react-native'
// condition (→ CJS index.js) instead of falling through to 'import' (→ ESM).
config.resolver.unstable_conditionNames = ['react-native', 'require', 'default'];

module.exports = config;
