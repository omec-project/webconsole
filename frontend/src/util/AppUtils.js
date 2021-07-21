// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

class AppUtils {

  static dashUuid(uuid) {
    let result = '';

    try {
      uuid = uuid.trim();
      result = uuid.substr(0, 8) + '-' + uuid.substr(8, 4) + '-' +
               uuid.substr(12, 4) + '-' + uuid.substr(16, 4) + '-' +
               uuid.substr(20);
    } catch (error) {}

    return result.length === 36 ? result : null;
  }

  static undashUuid(uuid) {
    let result = '';

    try {
      result = uuid.trim().replace(new RegExp('-', 'g'), '');
    } catch (error) {}

    return result.length === 32 ? result : null;
  }

  static wait(period) {
    return new Promise(resolve => setTimeout(resolve, period));
  }

}

export default AppUtils;
