// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

export default class Serializable {

  serialize() {
    return JSON.stringify(this);
  }

  /**
   * @param json {string}
   * @return {*} class extends Serializable
   */
  static deserialize(json) {
    try {
      let instance = JSON.parse(json);
      instance.__proto__ = this.prototype;
      instance.cvtAttrsFromString();
      return instance;
    } catch (error) {
      return null;
    }
  }

  cvtAttrsFromString() {
    // This method should be overrided by sub-classes if needed
  }

}
