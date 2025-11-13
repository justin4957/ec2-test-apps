const path = require('path');

module.exports = {
  mode: 'production',
  entry: './src/index.mjs',
  output: {
    filename: 'solid-client-bundle.js',
    path: path.resolve(__dirname, 'dist'),
  },
  module: {
    rules: [
      {
        test: /\.m?js$/,
        type: 'javascript/auto',
        resolve: {
          fullySpecified: false
        }
      }
    ]
  },
  resolve: {
    fallback: {
      "crypto": false,
      "stream": false,
      "buffer": false,
      "url": false,
      "util": false,
      "http": false,
      "https": false,
      "zlib": false,
      "assert": false
    }
  },
  performance: {
    hints: false,
    maxAssetSize: 512000,
    maxEntrypointSize: 512000
  }
};
