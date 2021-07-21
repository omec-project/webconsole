// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

export default class authActions {
  static SET_USER = 'AUTH/SET_USER';
  static LOGOUT = 'AUTH/LOGOUT';

  /**
   * @param user  {User}
   */
  static setUser(user) {
    return {
      type: this.SET_USER,
      user: user,
    };
  }

  static logout() {
    return {
      type: this.LOGOUT,
    };
  }
}
