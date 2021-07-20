// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

export default class subscriberActions {
  static SET_SUBSCRIBERS = 'SUBSCRIBER/SET_SUBSCRIBERS';

  /**
   * @param subscribers  {Subscriber}
   * //Bajo 20200710
   * @param subscribers  {subscriberData}
   */
  static setSubscribers(subscribers) {
    return {
      type: this.SET_SUBSCRIBERS,
      subscribers: subscribers,
    };
  }
}
