// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

import {createStore, applyMiddleware, combineReducers} from 'redux';
import createLogger from 'redux-logger';
import thunk from 'redux-thunk';
import reducers from '../redux/reducers';

export default function configureStore() {
  if (process.env.NODE_ENV !== 'production') {
    return createStore(
      combineReducers({
        ...reducers
      }),
      {},
      applyMiddleware(thunk, createLogger)
    );
  } else {
    return createStore(
      combineReducers({
        ...reducers
      }),
      {},
      applyMiddleware(thunk)
    );
  }
}
