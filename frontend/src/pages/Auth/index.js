// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

import React from 'react';
import {Route, Switch} from 'react-router-dom';
import Login from './Login';

const Auth = ({isLoggedIn}) => {
  if (isLoggedIn) {
    return null;
  }

  return (
    <div className="wrapper">
      <Switch>
        <Route component={Login}/>
      </Switch>
    </div>
  )
};

export default Auth;
