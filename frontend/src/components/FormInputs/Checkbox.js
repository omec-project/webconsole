// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

/* eslint-disable no-unused-vars */
import React, { Component } from 'react';
import cx from 'classnames';
import uncheckImage from 'assets/images/checkbox-1.svg';
import checkImage from 'assets/images/checkbox-2.svg';

class Checkbox extends Component {

  render() {
    let {
      input,
      label,
      type,
      meta: { touched, error, warning },
      disabled
    } = this.props;

    return (
      <label className={cx("checkbox", {
        checked: input.checked,
        disabled: disabled
      })}>
        <span className="icons">
          <img className="first-icon" src={uncheckImage} width={17} alt="" />
          <img className="second-icon" src={checkImage} width={17} alt="" />
        </span>
        <input {...input} type="checkbox" data-toggle="checkbox" disabled={disabled} />
        {label}
      </label>
    );
  }
}

export default Checkbox;
