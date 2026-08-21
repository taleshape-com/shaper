// SPDX-License-Identifier: MPL-2.0

const { getBinaryPath } = require('./bin/get-binary');

module.exports = {
  get path() {
    return getBinaryPath();
  },
  getBinaryPath,
};