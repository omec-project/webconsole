// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

import actions from '../actions/subscriberActions';

const initialState = {
  subscribers: [],
  subscribersMap: {}
};

export default function reducer(state = initialState, action) {
  let nextState = {...state};

  switch (action.type) {
    case actions.SET_SUBSCRIBERS:
      nextState.subscribers = action.subscribers;
      nextState.subscribersMap = createSubscribersMap(action.subscribers);
      return nextState;

    default:
      return state;
  }
}

function createSubscribersMap(subscribers) {
  let subscribersMap = {};
  subscribers.forEach(subscribers => subscribersMap[subscribers['ueId']] = subscribers);
  return subscribersMap;
}
