// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

import React, {Component} from 'react';
import {withRouter} from 'react-router-dom';
import {connect} from 'react-redux';
import Nav from './Nav';
import Free5gcLogo from "../../assets/images/free5gc_logo.png";

class SideBar extends Component {

  state = {};

  render() {
    return (
      <div className="sidebar">

        <div className="brand">
          <a href="/" className="brand-name">
            <img src={Free5gcLogo} alt="logo" className="logo"/>
          </a>

        </div>

        <div className="sidebar-wrapper">
          {/*<UserInfo/>*/}
          {/*<div className="line"/>*/}
          <Nav/>
        </div>
      </div>
    )
  }
}

const mapStateToProps = state => ({});

export default withRouter(connect(mapStateToProps)(SideBar));
