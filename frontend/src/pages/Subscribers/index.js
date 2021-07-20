// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

import React from 'react';
import {Route} from 'react-router-dom';
import SubscriberOverview from "./SubscriberOverview";

const Subscribers = ({match}) => (
  //console.log("In Subscribers")
  <div className="content">
    <Route exact path={`${match.url}/`} component={SubscriberOverview} />
    {/*<Route path={`${match.url}/:uuid`} component={} />*/}
  </div>
);

export default Subscribers;
