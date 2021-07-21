// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

import React from 'react';
import TextInput from './TextInput';
import Checkbox from './Checkbox';
import Radio from './Radio';

const renderField = (props) => (
  <div>
    { (props.type === 'email' ||
       props.type === 'password' ||
       props.type === 'text' ||
       props.type === 'number') &&
      <TextInput {...props} />
    }
    { props.type === 'checkbox' && <Checkbox {...props} /> }
    { props.type === 'radio' && <Radio {...props} /> }
  </div>
);

export default renderField;