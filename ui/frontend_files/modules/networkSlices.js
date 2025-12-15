// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

import { BaseManager } from './baseManager.js';
import { API_BASE } from '../app.js';

export class NetworkSliceManager extends BaseManager {
    constructor() {
        super('/network-slice', 'network-slices-list');
        this.type = 'network-slice';
        this.displayName = 'Network Slice';
    }

    // Override loadData to fetch complete network slice details
    async loadData() {
        try {
            this.showLoading();
            
            // First, get the list of network slice names
            const response = await fetch(`${API_BASE}${this.apiEndpoint}`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const sliceNames = await response.json();
            console.log('Network slice names:', sliceNames);
            
            // Check if we got valid data
            if (!Array.isArray(sliceNames)) {
                console.error('Expected array of slice names, got:', sliceNames);
                this.showError('Invalid response format from server');
                return;
            }
            
            // If no slices, show empty state
            if (sliceNames.length === 0) {
                this.data = [];
                this.render([]);
                return;
            }
            
            // Then, fetch complete details for each slice
            const sliceDetails = [];
            for (const sliceName of sliceNames) {
                try {
                    if (typeof sliceName !== 'string') {
                        console.warn('Invalid slice name:', sliceName);
                        continue;
                    }
                    
                    const detailResponse = await fetch(`${API_BASE}${this.apiEndpoint}/${encodeURIComponent(sliceName)}`);
                    if (detailResponse.ok) {
                        const sliceDetail = await detailResponse.json();
                        sliceDetails.push(sliceDetail);
                    } else {
                        console.warn(`Failed to load details for slice ${sliceName}: ${detailResponse.status}`);
                    }
                } catch (error) {
                    console.error(`Failed to load details for slice ${sliceName}:`, error);
                }
            }
            
            console.log('Complete network slice details:', sliceDetails);
            
            this.data = sliceDetails;
            this.render(sliceDetails);
            
        } catch (error) {
            this.showError(`Failed to load network slices: ${error.message}`);
            console.error('Load network slices error:', error);
        }
    }

    render(slices) {
        const container = document.getElementById(this.containerId);
        
        if (!slices || slices.length === 0) {
            this.showEmpty('No network slices found');
            return;
        }
        
        let html = '<div class="table-responsive"><table class="table table-striped">';
        html += '<thead><tr><th>Slice Name</th><th>SST</th><th>SD</th><th>Site</th><th>Device Groups</th><th>Actions</th></tr></thead><tbody>';
        
        slices.forEach(slice => {
            const sliceName = slice['slice-name'] || 'N/A';
            const sst = slice['slice-id']?.sst || 'N/A';
            const sd = slice['slice-id']?.sd || 'N/A';
            const siteName = slice['site-info']?.['site-name'] || 'N/A';
            const deviceGroups = slice['site-device-group'] || [];
            const gNodeBs = slice['site-info']?.gNodeBs || [];
            const appRules = slice['application-filtering-rules'] || [];
            
            html += `
                <tr class="network-slice-row" onclick="showNetworkSliceDetails('${sliceName}')" style="cursor: pointer;">
                    <td><strong>${sliceName}</strong></td>
                    <td><span class="badge bg-primary">${sst}</span></td>
                    <td><code>${sd}</code></td>
                    <td>${siteName}</td>
                    <td>
                        <span class="badge bg-secondary">${deviceGroups.length} groups</span>
                        ${deviceGroups.length > 0 ? `<br><small class="text-muted">${deviceGroups.join(', ')}</small>` : ''}
                        <br><small class="text-info">${gNodeBs.length} gNodeBs, ${appRules.length} rules</small>
                    </td>
                    <td onclick="event.stopPropagation();">
                        <button class="btn btn-sm btn-outline-primary me-1" 
                                onclick="editItem('${this.type}', '${sliceName}')">
                            <i class="fas fa-edit"></i> Edit
                        </button>
                        <button class="btn btn-sm btn-outline-danger" 
                                onclick="deleteItem('${this.type}', '${sliceName}')">
                            <i class="fas fa-trash"></i> Delete
                        </button>
                    </td>
                </tr>
            `;
        });
        
        html += '</tbody></table></div>';
        container.innerHTML = html;
    }

    getFormFields(isEdit = false) {
        return `
            <div class="mb-3">
                <label class="form-label">Slice Name</label>
                <input type="text" class="form-control" id="slice_name" 
                       ${isEdit ? 'readonly' : ''} required>
            </div>
            
            <h6 class="mt-4 mb-3">Slice ID Configuration</h6>
            <div class="row">
                <div class="col-md-6">
                    <div class="mb-3">
                        <label class="form-label">SST (Slice Service Type)</label>
                        <input type="text" class="form-control" id="sst" 
                               placeholder="e.g., 1" required>
                        <div class="form-text">Values: 1=eMBB, 2=URLLC, 3=mMTC, 4=Custom</div>
                    </div>
                </div>
                <div class="col-md-6">
                    <div class="mb-3">
                        <label class="form-label">SD (Slice Differentiator)</label>
                        <input type="text" class="form-control" id="sd" 
                               placeholder="e.g., 000001 (6 hex digits)" 
                               pattern="[0-9A-Fa-f]{6}" maxlength="6">
                        <div class="form-text">Optional: 6 hexadecimal digits</div>
                    </div>
                </div>
            </div>

            <h6 class="mt-4 mb-3">Site Information</h6>
            <div class="mb-3">
                <label class="form-label">Site Name</label>
                <input type="text" class="form-control" id="site_name" 
                       placeholder="e.g., site-1" required>
            </div>

            <div class="row">
                <div class="col-md-6">
                    <div class="mb-3">
                        <label class="form-label">MCC (Mobile Country Code)</label>
                        <input type="text" class="form-control" id="mcc" 
                               placeholder="e.g., 001" pattern="[0-9]{3}" maxlength="3" required>
                    </div>
                </div>
                <div class="col-md-6">
                    <div class="mb-3">
                        <label class="form-label">MNC (Mobile Network Code)</label>
                        <input type="text" class="form-control" id="mnc" 
                               placeholder="e.g., 01" pattern="[0-9]{2,3}" maxlength="3" required>
                    </div>
                </div>
            </div>

            <h6 class="mt-4 mb-3">Device Groups</h6>
            <div class="mb-3">
                <label class="form-label">Site Device Groups</label>
                <select class="form-select" id="site_device_group" multiple>
                    <option value="">Select device groups...</option>
                </select>
                <div class="form-text">Hold Ctrl/Cmd to select multiple groups</div>
            </div>

            <h6 class="mt-4 mb-3">gNodeB Configuration</h6>
            <div id="gnb-container">
                <div class="gnb-entry row mb-3">
                    <div class="col-md-5">
                        <div class="mb-3">
                            <label class="form-label">gNodeB Name</label>
                            <input type="text" class="form-control gnb-name" 
                                   placeholder="e.g., gnb-1" required>
                        </div>
                    </div>
                    <div class="col-md-5">
                        <div class="mb-3">
                            <label class="form-label">gNodeB TAC</label>
                            <input type="number" class="form-control gnb-tac" 
                                   placeholder="e.g., 1" min="1" max="16777215" required>
                        </div>
                    </div>
                    <div class="col-md-2 d-flex align-items-end">
                        <button type="button" class="btn btn-outline-danger mb-3" onclick="removeGnb(this)">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </div>
            </div>
            <button type="button" class="btn btn-outline-primary btn-sm mb-3" onclick="addGnb()">
                <i class="fas fa-plus"></i> Add gNodeB
            </button>

            <h6 class="mt-4 mb-3">UPF Configuration</h6>
            <div id="upf-container">
                <div class="upf-entry row mb-3">
                    <div class="col-md-8">
                        <div class="mb-3">
                            <label class="form-label">UPF Name</label>
                            <input type="text" class="form-control upf-name" 
                                   placeholder="e.g., upf-1.example.com">
                        </div>
                    </div>
                    <div class="col-md-2">
                        <div class="mb-3">
                            <label class="form-label">UPF Port</label>
                            <input type="number" class="form-control upf-port" 
                                   placeholder="8805" min="1" max="65535">
                        </div>
                    </div>
                    <div class="col-md-2 d-flex align-items-end">
                        <button type="button" class="btn btn-outline-danger mb-3" onclick="removeUpf(this)">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </div>
            </div>
            <button type="button" class="btn btn-outline-primary btn-sm mb-3" onclick="addUpf()">
                <i class="fas fa-plus"></i> Add UPF
            </button>

            <h6 class="mt-4 mb-3">Application Filtering Rules</h6>
            <div id="app-rules-container">
                <!-- Default rule will be added automatically by backend if empty -->
            </div>
            <button type="button" class="btn btn-outline-primary btn-sm mb-3" onclick="addApplicationRule()">
                <i class="fas fa-plus"></i> Add Application Rule
            </button>
            <div class="form-text mb-3">If no rules are specified, a default 'permit any' rule will be created automatically.</div>
        `;
    }

    validateFormData(data) {
        const errors = [];
        
        if (!data.slice_name || String(data.slice_name).trim() === '') {
            errors.push('Slice name is required');
        }
        
        if (!data.sst || String(data.sst).trim() === '') {
            errors.push('SST (Slice Service Type) is required');
        }
        
        if (data.sd && !/^[0-9A-Fa-f]{6}$/.test(String(data.sd))) {
            errors.push('SD must be exactly 6 hexadecimal digits (e.g., 000001)');
        }
        
        if (!data.site_name || String(data.site_name).trim() === '') {
            errors.push('Site name is required');
        }
        
        if (!data.mcc || !/^[0-9]{3}$/.test(String(data.mcc))) {
            errors.push('MCC must be exactly 3 digits');
        }
        
        if (!data.mnc || !/^[0-9]{2,3}$/.test(String(data.mnc))) {
            errors.push('MNC must be 2 or 3 digits');
        }
        
        // Validate gNodeBs collected from form
        const gNodeBs = this.collectGnbData();
        if (gNodeBs.length === 0) {
            errors.push('At least one gNodeB is required');
        } else {
            gNodeBs.forEach((gnb, index) => {
                if (!gnb.name || String(gnb.name).trim() === '') {
                    errors.push(`gNodeB ${index + 1}: Name is required`);
                }
                if (!gnb.tac || isNaN(gnb.tac) || gnb.tac < 1 || gnb.tac > 16777215) {
                    errors.push(`gNodeB ${index + 1}: TAC must be between 1 and 16777215`);
                }
            });
        }
        
        return {
            isValid: errors.length === 0,
            errors: errors
        };
    }

    preparePayload(formData, isEdit = false) {
        // Prepare site device groups array
        const siteDeviceGroups = [];
        if (formData.site_device_group) {
            // If multiple values selected
            if (Array.isArray(formData.site_device_group)) {
                siteDeviceGroups.push(...formData.site_device_group.filter(g => g));
            } else if (formData.site_device_group.trim() !== '') {
                siteDeviceGroups.push(formData.site_device_group);
            }
        }

        // Prepare gNodeBs array - use data from edit form if in edit mode
        const gNodeBs = isEdit && formData.gNodeBs ? formData.gNodeBs : this.collectGnbData();
        
        // Prepare application filtering rules - use data from edit form if in edit mode
        const appRules = isEdit && formData.applicationRules ? formData.applicationRules : this.collectApplicationRules();

        // Prepare UPF object - use data from edit form if in edit mode
        const upf = isEdit && formData.upf ? formData.upf : this.collectUpfData();

        return {
            "slice-name": formData.slice_name,
            "slice-id": {
                "sst": formData.sst,
                "sd": formData.sd || ""
            },
            "site-device-group": siteDeviceGroups,
            "site-info": {
                "site-name": formData.site_name,
                "plmn": {
                    "mcc": formData.mcc,
                    "mnc": formData.mnc
                },
                "gNodeBs": gNodeBs,
                "upf": upf
            },
            "application-filtering-rules": appRules
        };
    }

    // Override createItem to include slice name in URL for network slices
    async createItem(itemData) {
        try {
            const sliceName = itemData['slice-name'];
            const response = await fetch(`${API_BASE}${this.apiEndpoint}/${sliceName}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(itemData)
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(errorText || `HTTP ${response.status}`);
            }

            return await response.json();
        } catch (error) {
            throw error;
        }
    }

    async loadDeviceGroups() {
        try {
            const response = await fetch(`${API_BASE}/device-group`);
            if (response.ok) {
                const deviceGroupNames = await response.json();
                const select = document.getElementById('site_device_group');
                if (select && Array.isArray(deviceGroupNames)) {
                    select.innerHTML = '<option value="">Select device groups...</option>';
                    deviceGroupNames.forEach(groupName => {
                        if (typeof groupName === 'string') {
                            const option = document.createElement('option');
                            option.value = groupName;
                            option.textContent = groupName;
                            select.appendChild(option);
                        }
                    });
                }
            }
        } catch (error) {
            console.warn('Failed to load device groups:', error.message);
        }
    }

    async loadItemData(name) {
        try {
            const response = await fetch(`${API_BASE}${this.apiEndpoint}/${encodeURIComponent(name)}`);
            if (response.ok) {
                const data = await response.json();
                
                // Populate basic fields
                this.setFieldValue('slice_name', data['slice-name']);
                this.setFieldValue('sst', data['slice-id']?.sst);
                this.setFieldValue('sd', data['slice-id']?.sd);
                
                // Populate site info
                const siteInfo = data['site-info'] || {};
                this.setFieldValue('site_name', siteInfo['site-name']);
                this.setFieldValue('mcc', siteInfo.plmn?.mcc);
                this.setFieldValue('mnc', siteInfo.plmn?.mnc);
                
                // Populate device groups
                const deviceGroups = data['site-device-group'] || [];
                const select = document.getElementById('site_device_group');
                if (select && deviceGroups.length > 0) {
                    Array.from(select.options).forEach(option => {
                        option.selected = deviceGroups.includes(option.value);
                    });
                }
                
                // Populate multiple gNodeBs
                const gNodeBs = siteInfo.gNodeBs || [];
                this.loadGnbData(gNodeBs);
                
                // Populate UPF info
                const upf = siteInfo.upf || {};
                this.loadUpfData(upf);
                
                // Populate application filtering rules
                const appRules = data['application-filtering-rules'] || [];
                this.loadApplicationRules(appRules);
            }
        } catch (error) {
            console.error('Failed to load item data:', error);
        }
    }

    setFieldValue(fieldId, value) {
        const field = document.getElementById(fieldId);
        if (field && value !== undefined && value !== null) {
            field.value = value;
        }
    }

    // Override the base method to load device groups when form is shown
    async showCreateForm() {
        await super.showCreateForm();
        await this.loadDeviceGroups();
    }

    async showEditForm(name) {
        await super.showEditForm(name);
        await this.loadDeviceGroups();
    }

    // New methods for details view
    async showDetails(sliceName) {
        try {
            const response = await fetch(`${API_BASE}${this.apiEndpoint}/${encodeURIComponent(sliceName)}`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const sliceData = await response.json();
            this.currentSliceData = sliceData;
            this.currentSliceName = sliceName;
            this.renderDetailsView(sliceData);
            
        } catch (error) {
            console.error('Failed to load network slice details:', error);
            // Show error notification
            window.app?.notificationManager?.showNotification('Error loading network slice details', 'error');
        }
    }

    renderDetailsView(sliceData) {
        const container = document.getElementById('network-slice-details-content');
        const title = document.getElementById('network-slice-detail-title');
        
        if (!container || !title) {
            console.error('Details container not found');
            return;
        }

        const sliceName = sliceData['slice-name'] || 'Unknown';
        title.textContent = `Network Slice: ${sliceName}`;

        const html = `
            <div id="network-slice-details-view-mode">
                ${this.renderReadOnlyDetails(sliceData)}
            </div>
            <div id="network-slice-details-edit-mode" style="display: none;">
                ${this.renderEditableDetails(sliceData)}
            </div>
        `;

        container.innerHTML = html;
    }

    renderReadOnlyDetails(sliceData) {
        const siteInfo = sliceData['site-info'] || {};
        const plmn = siteInfo.plmn || {};
        const gNodeBs = siteInfo.gNodeBs || [];
        const upf = siteInfo.upf || {};
        const deviceGroups = sliceData['site-device-group'] || [];

        return `
            <div class="row">
                <div class="col-md-6">
                    <div class="card mb-3">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-layer-group me-2"></i>Slice Information</h6>
                        </div>
                        <div class="card-body">
                            <div class="mb-2">
                                <strong>Slice Name:</strong> ${sliceData['slice-name'] || 'N/A'}
                            </div>
                            <div class="mb-2">
                                <strong>SST (Slice Service Type):</strong> 
                                <span class="badge bg-primary ms-1">${sliceData['slice-id']?.sst || 'N/A'}</span>
                            </div>
                            <div class="mb-2">
                                <strong>SD (Slice Differentiator):</strong> 
                                <code class="ms-1">${sliceData['slice-id']?.sd || 'Not specified'}</code>
                            </div>
                        </div>
                    </div>

                    <div class="card mb-3">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-map-marker-alt me-2"></i>Site Information</h6>
                        </div>
                        <div class="card-body">
                            <div class="mb-2">
                                <strong>Site Name:</strong> ${siteInfo['site-name'] || 'N/A'}
                            </div>
                            <div class="mb-2">
                                <strong>MCC:</strong> <code>${plmn.mcc || 'N/A'}</code>
                            </div>
                            <div class="mb-2">
                                <strong>MNC:</strong> <code>${plmn.mnc || 'N/A'}</code>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="col-md-6">
                    <div class="card mb-3">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-mobile-alt me-2"></i>Device Groups</h6>
                        </div>
                        <div class="card-body">
                            <div class="mb-2">
                                <strong>Total Groups:</strong> <span class="badge bg-secondary">${deviceGroups.length}</span>
                            </div>
                            ${deviceGroups.length > 0 ? `
                                <div class="mb-2">
                                    <strong>Groups:</strong>
                                    <div class="mt-2">
                                        ${deviceGroups.map(group => `<span class="badge bg-light text-dark me-1 mb-1">${group}</span>`).join('')}
                                    </div>
                                </div>
                            ` : '<p class="text-muted">No device groups assigned</p>'}
                        </div>
                    </div>

                    <div class="card mb-3">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-tower-broadcast me-2"></i>gNodeBs (${gNodeBs.length})</h6>
                        </div>
                        <div class="card-body">
                            ${gNodeBs.length > 0 ? `
                                ${gNodeBs.map((gnb, index) => `
                                    <div class="mb-3 ${index < gNodeBs.length - 1 ? 'border-bottom pb-2' : ''}">
                                        <div class="row">
                                            <div class="col-md-6">
                                                <strong>Name:</strong> ${gnb.name || 'N/A'}
                                            </div>
                                            <div class="col-md-6">
                                                <strong>TAC:</strong> <code>${gnb.tac || 'N/A'}</code>
                                            </div>
                                        </div>
                                    </div>
                                `).join('')}
                            ` : '<p class="text-muted">No gNodeBs configured</p>'}
                        </div>
                    </div>

                    <div class="card mb-3">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-server me-2"></i>UPF Configuration</h6>
                        </div>
                        <div class="card-body">
                            ${Object.keys(upf).length > 0 ? `
                                <div class="table-responsive">
                                    <table class="table table-sm">
                                        <thead>
                                            <tr>
                                                <th>UPF Name</th>
                                                <th>Port</th>
                                                <th>Status</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            ${Object.entries(upf).map(([upfName, upfConfig]) => `
                                                <tr>
                                                    <td><strong>${upfName}</strong></td>
                                                    <td><code>${upfConfig['upf-port'] || 'N/A'}</code></td>
                                                    <td><span class="badge bg-success">Active</span></td>
                                                </tr>
                                            `).join('')}
                                        </tbody>
                                    </table>
                                </div>
                            ` : '<p class="text-muted">No UPF configured</p>'}
                        </div>
                    </div>
                </div>
            </div>

            <div class="row">
                <div class="col-12">
                    <div class="card mb-3">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-filter me-2"></i>Application Filtering Rules (${(sliceData['application-filtering-rules'] || []).length})</h6>
                        </div>
                        <div class="card-body">
                            ${(sliceData['application-filtering-rules'] || []).length > 0 ? `
                                <div class="table-responsive">
                                    <table class="table table-sm table-striped">
                                        <thead class="table-dark">
                                            <tr>
                                                <th>Rule Name</th>
                                                <th>Priority</th>
                                                <th>Action</th>
                                                <th>Endpoint</th>
                                                <th>Protocol</th>
                                                <th>Port Range</th>
                                                <th>Bitrate</th>
                                                <th>Traffic Class</th>
                                                <th>Trigger</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            ${sliceData['application-filtering-rules'].map((rule, index) => `
                                                <tr>
                                                    <td><strong>${rule['rule-name'] || `Rule-${index + 1}`}</strong></td>
                                                    <td><span class="badge bg-primary">${rule.priority !== undefined ? rule.priority : 'N/A'}</span></td>
                                                    <td>
                                                        <span class="badge ${rule.action === 'permit' ? 'bg-success' : rule.action === 'deny' ? 'bg-danger' : 'bg-secondary'}">
                                                            ${rule.action || 'N/A'}
                                                        </span>
                                                    </td>
                                                    <td><code>${rule.endpoint || 'any'}</code></td>
                                                    <td>${rule.protocol !== undefined ? `<code>${rule.protocol}</code>` : 'any'}</td>
                                                    <td>
                                                        ${rule['dest-port-start'] !== undefined || rule['dest-port-end'] !== undefined ? 
                                                            `<code>${rule['dest-port-start'] || 'any'} - ${rule['dest-port-end'] || 'any'}</code>` : 
                                                            '<span class="text-muted">any</span>'
                                                        }
                                                    </td>
                                                    <td>
                                                        ${rule['app-mbr-uplink'] !== undefined || rule['app-mbr-downlink'] !== undefined ? `
                                                            <div class="small">
                                                                ${rule['app-mbr-uplink'] !== undefined ? `<div>↑ ${rule['app-mbr-uplink']} ${rule['bitrate-unit'] || 'bps'}</div>` : ''}
                                                                ${rule['app-mbr-downlink'] !== undefined ? `<div>↓ ${rule['app-mbr-downlink']} ${rule['bitrate-unit'] || 'bps'}</div>` : ''}
                                                            </div>
                                                        ` : '<span class="text-muted">unlimited</span>'}
                                                    </td>
                                                    <td>
                                                        ${rule['traffic-class'] ? `
                                                            <div class="small">
                                                                <div><strong>${rule['traffic-class'].name || 'N/A'}</strong></div>
                                                                ${rule['traffic-class'].qci !== undefined ? `<div>QCI: ${rule['traffic-class'].qci}</div>` : ''}
                                                                ${rule['traffic-class'].arp !== undefined ? `<div>ARP: ${rule['traffic-class'].arp}</div>` : ''}
                                                                ${rule['traffic-class'].pdb !== undefined ? `<div>PDB: ${rule['traffic-class'].pdb}ms</div>` : ''}
                                                                ${rule['traffic-class'].pelr !== undefined ? `<div>PELR: ${rule['traffic-class'].pelr}</div>` : ''}
                                                            </div>
                                                        ` : '<span class="text-muted">default</span>'}
                                                    </td>
                                                    <td>
                                                        ${rule['rule-trigger'] ? `<code>${rule['rule-trigger']}</code>` : '<span class="text-muted">auto</span>'}
                                                    </td>
                                                </tr>
                                            `).join('')}
                                        </tbody>
                                    </table>
                                </div>
                                <div class="mt-3 p-2 bg-light rounded">
                                    <small class="text-muted">
                                        <strong>Legend:</strong> 
                                        Priority (0=lowest, higher numbers = higher priority) | 
                                        Actions: <span class="badge bg-success">permit</span> <span class="badge bg-danger">deny</span> | 
                                        Protocol: TCP=6, UDP=17, ICMP=1 | 
                                        Bitrate units: bps, Kbps, Mbps, Gbps
                                    </small>
                                </div>
                            ` : `
                                <div class="text-center p-4">
                                    <i class="fas fa-filter fa-3x text-muted mb-3"></i>
                                    <p class="text-muted">No application filtering rules configured</p>
                                    <small class="text-muted">When no rules are specified, a default 'permit any' rule is automatically applied</small>
                                </div>
                            `}
                        </div>
                    </div>
                </div>
            </div>

            <div class="row">
                <div class="col-12">
                    <div class="card">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-info-circle me-2"></i>Technical Information</h6>
                        </div>
                        <div class="card-body">
                            <div class="bg-light p-3 rounded">
                                <div class="row">
                                    <div class="col-md-3">
                                        <small class="text-muted">SST Values:</small>
                                        <div><strong>1=eMBB, 2=URLLC, 3=mMTC, 4=Custom</strong></div>
                                    </div>
                                    <div class="col-md-3">
                                        <small class="text-muted">SD Format:</small>
                                        <div><strong>6 hexadecimal digits</strong></div>
                                    </div>
                                    <div class="col-md-3">
                                        <small class="text-muted">MCC/MNC:</small>
                                        <div><strong>Country/Network Codes</strong></div>
                                    </div>
                                    <div class="col-md-3">
                                        <small class="text-muted">Status:</small>
                                        <div>
                                            <span class="badge bg-success">
                                                <i class="fas fa-check-circle me-1"></i>Active
                                            </span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    renderEditableDetails(sliceData) {
        const siteInfo = sliceData['site-info'] || {};
        const plmn = siteInfo.plmn || {};
        const gNodeBs = siteInfo.gNodeBs || [];
        const upf = siteInfo.upf || {};
        const deviceGroups = sliceData['site-device-group'] || [];

        return `
            <form id="networkSliceDetailsEditForm">
                <div class="row">
                    <div class="col-md-6">
                        <div class="card mb-3">
                            <div class="card-header">
                                <h6 class="mb-0"><i class="fas fa-edit me-2"></i>Edit Slice Information</h6>
                            </div>
                            <div class="card-body">
                                <div class="mb-3">
                                    <label class="form-label">Slice Name</label>
                                    <input type="text" class="form-control" id="edit_slice_name" 
                                           value="${sliceData['slice-name'] || ''}" readonly>
                                    <div class="form-text">Slice name cannot be changed</div>
                                </div>
                                <div class="row">
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">SST (Slice Service Type)</label>
                                            <input type="text" class="form-control" id="edit_sst" 
                                                   value="${sliceData['slice-id']?.sst || ''}" placeholder="e.g., 1" required>
                                            <div class="form-text">1=eMBB, 2=URLLC, 3=mMTC, 4=Custom</div>
                                        </div>
                                    </div>
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">SD (Slice Differentiator)</label>
                                            <input type="text" class="form-control" id="edit_sd" 
                                                   value="${sliceData['slice-id']?.sd || ''}" placeholder="e.g., 000001" 
                                                   pattern="[0-9A-Fa-f]{6}" maxlength="6">
                                            <div class="form-text">6 hexadecimal digits</div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="card mb-3">
                            <div class="card-header">
                                <h6 class="mb-0"><i class="fas fa-map-marker-alt me-2"></i>Site Information</h6>
                            </div>
                            <div class="card-body">
                                <div class="mb-3">
                                    <label class="form-label">Site Name</label>
                                    <input type="text" class="form-control" id="edit_site_name" 
                                           value="${siteInfo['site-name'] || ''}" placeholder="e.g., site-1" required>
                                </div>
                                <div class="row">
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">MCC</label>
                                            <input type="text" class="form-control" id="edit_mcc" 
                                                   value="${plmn.mcc || ''}" placeholder="e.g., 001" 
                                                   pattern="[0-9]{3}" maxlength="3" required>
                                        </div>
                                    </div>
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">MNC</label>
                                            <input type="text" class="form-control" id="edit_mnc" 
                                                   value="${plmn.mnc || ''}" placeholder="e.g., 01" 
                                                   pattern="[0-9]{2,3}" maxlength="3" required>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="col-md-6">
                        <div class="card mb-3">
                            <div class="card-header">
                                <h6 class="mb-0"><i class="fas fa-mobile-alt me-2"></i>Device Groups</h6>
                            </div>
                            <div class="card-body">
                                <div class="mb-3">
                                    <label class="form-label">Site Device Groups</label>
                                    <select class="form-select" id="edit_site_device_group" multiple>
                                        <option value="">Select device groups...</option>
                                    </select>
                                    <div class="form-text">Hold Ctrl/Cmd to select multiple groups</div>
                                </div>
                            </div>
                        </div>

                        <div class="card mb-3">
                            <div class="card-header">
                                <div class="d-flex justify-content-between align-items-center">
                                    <h6 class="mb-0"><i class="fas fa-tower-broadcast me-2"></i>gNodeB Configuration</h6>
                                    <button type="button" class="btn btn-sm btn-outline-primary" onclick="addGnb()">
                                        <i class="fas fa-plus"></i> Add gNodeB
                                    </button>
                                </div>
                            </div>
                            <div class="card-body">
                                <div id="gnb-container">
                                    <!-- gNodeBs will be loaded dynamically -->
                                </div>
                            </div>
                        </div>

                        <div class="card mb-3">
                            <div class="card-header">
                                <div class="d-flex justify-content-between align-items-center">
                                    <h6 class="mb-0"><i class="fas fa-server me-2"></i>UPF Configuration</h6>
                                    <button type="button" class="btn btn-sm btn-outline-primary" onclick="addUpf()">
                                        <i class="fas fa-plus"></i> Add UPF
                                    </button>
                                </div>
                            </div>
                            <div class="card-body">
                                <div id="upf-container">
                                    <!-- UPFs will be loaded dynamically -->
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                
                <div class="row">
                    <div class="col-12">
                        <div class="card mb-3">
                            <div class="card-header">
                                <div class="d-flex justify-content-between align-items-center">
                                    <h6 class="mb-0"><i class="fas fa-filter me-2"></i>Application Filtering Rules</h6>
                                    <button type="button" class="btn btn-sm btn-outline-primary" onclick="addApplicationRule()">
                                        <i class="fas fa-plus"></i> Add Rule
                                    </button>
                                </div>
                            </div>
                            <div class="card-body">
                                <div id="app-rules-container">
                                    <!-- Application rules will be loaded dynamically -->
                                </div>
                                <div class="form-text">If no rules are specified, a default 'permit any' rule will be created automatically.</div>
                            </div>
                        </div>
                    </div>
                </div>
                
                <div class="row">
                    <div class="col-12">
                        <div class="d-flex justify-content-end">
                            <button type="button" class="btn btn-secondary me-2" onclick="cancelNetworkSliceEdit()">Cancel</button>
                            <button type="button" class="btn btn-primary" onclick="saveNetworkSliceDetailsEdit()">Save Changes</button>
                        </div>
                    </div>
                </div>
            </form>
        `;
    }

    async saveEdit() {
        try {
            const formData = this.getEditFormData();
            const validation = this.validateFormData(formData);
            
            if (!validation.isValid) {
                window.app?.notificationManager?.showNotification(validation.errors.join('<br>'), 'error');
                return;
            }

            const payload = this.preparePayload(formData, true);
            await this.updateItem(this.currentSliceName, payload);
            
            // Refresh the details view
            await this.showDetails(this.currentSliceName);
            this.toggleEditMode(false);
            
            window.app?.notificationManager?.showNotification('Network slice updated successfully!', 'success');
            
        } catch (error) {
            console.error('Failed to save network slice:', error);
            window.app?.notificationManager?.showNotification(`Failed to save network slice: ${error.message}`, 'error');
        }
    }

    getEditFormData() {
        const deviceGroupSelect = document.getElementById('edit_site_device_group');
        const selectedGroups = Array.from(deviceGroupSelect.selectedOptions).map(option => option.value).filter(val => val);

        return {
            slice_name: document.getElementById('edit_slice_name')?.value || '',
            sst: document.getElementById('edit_sst')?.value || '',
            sd: document.getElementById('edit_sd')?.value || '',
            site_name: document.getElementById('edit_site_name')?.value || '',
            mcc: document.getElementById('edit_mcc')?.value || '',
            mnc: document.getElementById('edit_mnc')?.value || '',
            site_device_group: selectedGroups,
            // Collect gNodeBs data
            gNodeBs: this.collectGnbData(),
            // Collect UPF data
            upf: this.collectUpfData(),
            // Collect Application Filtering Rules data
            applicationRules: this.collectApplicationRules()
        };
    }

    async loadDeviceGroupsForEdit() {
        try {
            const response = await fetch(`${API_BASE}/device-group`);
            if (response.ok) {
                const deviceGroupNames = await response.json();
                const select = document.getElementById('edit_site_device_group');
                if (select && Array.isArray(deviceGroupNames)) {
                    select.innerHTML = '<option value="">Select device groups...</option>';
                    deviceGroupNames.forEach(groupName => {
                        if (typeof groupName === 'string') {
                            const option = document.createElement('option');
                            option.value = groupName;
                            option.textContent = groupName;
                            select.appendChild(option);
                        }
                    });

                    // Pre-select current device groups
                    const currentGroups = this.currentSliceData['site-device-group'] || [];
                    Array.from(select.options).forEach(option => {
                        option.selected = currentGroups.includes(option.value);
                    });
                }
            }
        } catch (error) {
            console.warn('Failed to load device groups:', error.message);
        }
    }

    toggleEditMode(enable = null) {
        const detailsView = document.getElementById('network-slice-details-view-mode');
        const editView = document.getElementById('network-slice-details-edit-mode');
        const editBtn = document.getElementById('edit-network-slice-btn');
        
        if (!detailsView || !editView || !editBtn) return;
        
        const isEditing = enable !== null ? enable : editView.style.display !== 'none';
        
        if (isEditing) {
            detailsView.style.display = 'block';
            editView.style.display = 'none';
            editBtn.innerHTML = '<i class="fas fa-edit me-1"></i>Edit';
        } else {
            detailsView.style.display = 'none';
            editView.style.display = 'block';
            editBtn.innerHTML = '<i class="fas fa-times me-1"></i>Cancel';
            
            // Load device groups and gNodeBs when entering edit mode
            this.loadDeviceGroupsForEdit();
            
            // Load current gNodeBs data into edit form
            const siteInfo = this.currentSliceData['site-info'] || {};
            const gNodeBs = siteInfo.gNodeBs || [];
            this.loadGnbData(gNodeBs);
            
            // Load current UPF data into edit form
            const upf = siteInfo.upf || {};
            this.loadUpfData(upf);
            
            // Load current Application Filtering Rules into edit form
            const appRules = this.currentSliceData['application-filtering-rules'] || [];
            this.loadApplicationRules(appRules);
        }
    }

    // Helper method to get current slice data
    getCurrentSliceData() {
        return this.currentSliceData;
    }

    async deleteFromDetails() {
        try {
            await this.deleteItem(this.currentSliceName);
            window.app?.notificationManager?.showNotification('Network slice deleted successfully!', 'success');
            
            // Navigate back to the list
            window.showSection('network-slices');
            
        } catch (error) {
            console.error('Failed to delete network slice:', error);
            window.app?.notificationManager?.showNotification(`Failed to delete network slice: ${error.message}`, 'error');
        }
    }

    // gNodeB Management Methods
    collectGnbData() {
        const gNodeBs = [];
        const gnbEntries = document.querySelectorAll('.gnb-entry');
        
        gnbEntries.forEach(entry => {
            const name = entry.querySelector('.gnb-name')?.value?.trim();
            const tac = entry.querySelector('.gnb-tac')?.value;
            
            if (name && tac) {
                gNodeBs.push({
                    "name": name,
                    "tac": parseInt(tac)
                });
            }
        });
        
        return gNodeBs;
    }

    loadGnbData(gNodeBs) {
        const container = document.getElementById('gnb-container');
        if (!container) return;
        
        container.innerHTML = '';
        
        if (gNodeBs.length === 0) {
            gNodeBs.push({ name: '', tac: '' }); // Add empty entry
        }
        
        gNodeBs.forEach((gnb, index) => {
            const gnbHtml = `
                <div class="gnb-entry row mb-3">
                    <div class="col-md-5">
                        <div class="mb-3">
                            <label class="form-label">gNodeB Name</label>
                            <input type="text" class="form-control gnb-name" 
                                   placeholder="e.g., gnb-${index + 1}" value="${gnb.name || ''}" required>
                        </div>
                    </div>
                    <div class="col-md-5">
                        <div class="mb-3">
                            <label class="form-label">gNodeB TAC</label>
                            <input type="number" class="form-control gnb-tac" 
                                   placeholder="e.g., ${index + 1}" min="1" max="16777215" 
                                   value="${gnb.tac || ''}" required>
                        </div>
                    </div>
                    <div class="col-md-2 d-flex align-items-end">
                        <button type="button" class="btn btn-outline-danger mb-3" onclick="removeGnb(this)">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </div>
            `;
            container.insertAdjacentHTML('beforeend', gnbHtml);
        });
    }

    // Application Rules Management Methods
    collectApplicationRules() {
        const rules = [];
        const ruleEntries = document.querySelectorAll('.app-rule-entry');
        
        ruleEntries.forEach(entry => {
            const ruleName = entry.querySelector('.rule-name')?.value?.trim();
            const priority = entry.querySelector('.rule-priority')?.value;
            const action = entry.querySelector('.rule-action')?.value;
            const endpoint = entry.querySelector('.rule-endpoint')?.value?.trim();
            const protocol = entry.querySelector('.rule-protocol')?.value;
            const startPort = entry.querySelector('.rule-start-port')?.value;
            const endPort = entry.querySelector('.rule-end-port')?.value;
            const ruleTrigger = entry.querySelector('.rule-trigger')?.value?.trim();
            const mbrUplink = entry.querySelector('.rule-mbr-uplink')?.value;
            const mbrDownlink = entry.querySelector('.rule-mbr-downlink')?.value;
            const bitrateUnit = entry.querySelector('.rule-bitrate-unit')?.value || 'bps';
            
            // Traffic class fields
            const tcName = entry.querySelector('.tc-name')?.value?.trim();
            const tcQci = entry.querySelector('.tc-qci')?.value;
            const tcArp = entry.querySelector('.tc-arp')?.value;
            const tcPdb = entry.querySelector('.tc-pdb')?.value;
            const tcPelr = entry.querySelector('.tc-pelr')?.value;
            
            if (ruleName && action && endpoint) {
                rules.push({
                    "rule-name": ruleName,
                    "priority": parseInt(priority) || 0,
                    "action": action,
                    "endpoint": endpoint,
                    "protocol": parseInt(protocol) || 0,
                    "dest-port-start": parseInt(startPort) || 0,
                    "dest-port-end": parseInt(endPort) || 65535,
                    "rule-trigger": ruleTrigger || "",
                    "app-mbr-uplink": parseInt(mbrUplink) || 0,
                    "app-mbr-downlink": parseInt(mbrDownlink) || 0,
                    "bitrate-unit": bitrateUnit,
                    "traffic-class": {
                        "name": tcName || "default",
                        "qci": parseInt(tcQci) || 9,
                        "arp": parseInt(tcArp) || 8,
                        "pdb": parseInt(tcPdb) || 100,
                        "pelr": parseInt(tcPelr) || 6
                    }
                });
            }
        });
        
        return rules;
    }

    loadApplicationRules(rules) {
        const container = document.getElementById('app-rules-container');
        if (!container) return;
        
        container.innerHTML = '';
        
        rules.forEach((rule, index) => {
            this.addApplicationRuleEntry(rule, index);
        });
    }

    addApplicationRuleEntry(rule = null, index = 0) {
        const container = document.getElementById('app-rules-container');
        if (!container) return;
        
        const ruleHtml = `
            <div class="app-rule-entry card mb-3">
                <div class="card-header">
                    <div class="d-flex justify-content-between align-items-center">
                        <h6 class="mb-0">Application Rule ${index + 1}</h6>
                        <button type="button" class="btn btn-sm btn-outline-danger" onclick="removeApplicationRule(this)">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </div>
                <div class="card-body">
                    <div class="row">
                        <div class="col-md-3">
                            <div class="mb-3">
                                <label class="form-label">Rule Name</label>
                                <input type="text" class="form-control rule-name" 
                                       placeholder="e.g., rule-${index + 1}" value="${rule?.['rule-name'] || ''}" required>
                            </div>
                        </div>
                        <div class="col-md-3">
                            <div class="mb-3">
                                <label class="form-label">Priority</label>
                                <input type="number" class="form-control rule-priority" 
                                       placeholder="0" min="0" value="${rule?.priority || 0}">
                                <div class="form-text">Higher number = higher priority</div>
                            </div>
                        </div>
                        <div class="col-md-3">
                            <div class="mb-3">
                                <label class="form-label">Action</label>
                                <select class="form-select rule-action" required>
                                    <option value="permit" ${rule?.action === 'permit' ? 'selected' : ''}>Permit</option>
                                    <option value="deny" ${rule?.action === 'deny' ? 'selected' : ''}>Deny</option>
                                </select>
                            </div>
                        </div>
                        <div class="col-md-3">
                            <div class="mb-3">
                                <label class="form-label">Endpoint</label>
                                <input type="text" class="form-control rule-endpoint" 
                                       placeholder="e.g., any or 192.168.1.0/24" value="${rule?.endpoint || 'any'}" required>
                            </div>
                        </div>
                    </div>
                    <div class="row">
                        <div class="col-md-3">
                            <div class="mb-3">
                                <label class="form-label">Protocol</label>
                                <input type="number" class="form-control rule-protocol" 
                                       placeholder="0 (any)" min="0" max="255" value="${rule?.protocol || 0}">
                            </div>
                        </div>
                        <div class="col-md-3">
                            <div class="mb-3">
                                <label class="form-label">Start Port</label>
                                <input type="number" class="form-control rule-start-port" 
                                       placeholder="0" min="0" max="65535" value="${rule?.['dest-port-start'] || 0}">
                            </div>
                        </div>
                        <div class="col-md-3">
                            <div class="mb-3">
                                <label class="form-label">End Port</label>
                                <input type="number" class="form-control rule-end-port" 
                                       placeholder="65535" min="0" max="65535" value="${rule?.['dest-port-end'] || 65535}">
                            </div>
                        </div>
                        <div class="col-md-3">
                            <div class="mb-3">
                                <label class="form-label">Rule Trigger</label>
                                <input type="text" class="form-control rule-trigger" 
                                       placeholder="Optional trigger" value="${rule?.['rule-trigger'] || ''}">
                            </div>
                        </div>
                    </div>
                    <div class="row">
                        <div class="col-md-4">
                            <div class="mb-3">
                                <label class="form-label">MBR Uplink (${rule?.['bitrate-unit'] || 'bps'})</label>
                                <input type="number" class="form-control rule-mbr-uplink" 
                                       placeholder="0" min="0" value="${rule?.['app-mbr-uplink'] || 0}">
                            </div>
                        </div>
                        <div class="col-md-4">
                            <div class="mb-3">
                                <label class="form-label">MBR Downlink (${rule?.['bitrate-unit'] || 'bps'})</label>
                                <input type="number" class="form-control rule-mbr-downlink" 
                                       placeholder="0" min="0" value="${rule?.['app-mbr-downlink'] || 0}">
                            </div>
                        </div>
                        <div class="col-md-4">
                            <div class="mb-3">
                                <label class="form-label">Bitrate Unit</label>
                                <select class="form-select rule-bitrate-unit">
                                    <option value="bps" ${rule?.['bitrate-unit'] === 'bps' ? 'selected' : ''}>bps</option>
                                    <option value="kbps" ${rule?.['bitrate-unit'] === 'kbps' ? 'selected' : ''}>Kbps</option>
                                    <option value="mbps" ${rule?.['bitrate-unit'] === 'mbps' ? 'selected' : ''}>Mbps</option>
                                    <option value="gbps" ${rule?.['bitrate-unit'] === 'gbps' ? 'selected' : ''}>Gbps</option>
                                </select>
                            </div>
                        </div>
                    </div>
                    <div class="card mt-3">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-cogs me-2"></i>Traffic Class Configuration</h6>
                        </div>
                        <div class="card-body">
                            <div class="row">
                                <div class="col-md-3">
                                    <div class="mb-3">
                                        <label class="form-label">Traffic Class Name</label>
                                        <input type="text" class="form-control tc-name" 
                                               placeholder="default" value="${rule?.['traffic-class']?.name || 'default'}" required>
                                    </div>
                                </div>
                                <div class="col-md-2">
                                    <div class="mb-3">
                                        <label class="form-label">QCI</label>
                                        <input type="number" class="form-control tc-qci" 
                                               placeholder="9" min="1" max="9" value="${rule?.['traffic-class']?.qci || 9}" required>
                                    </div>
                                </div>
                                <div class="col-md-2">
                                    <div class="mb-3">
                                        <label class="form-label">ARP</label>
                                        <input type="number" class="form-control tc-arp" 
                                               placeholder="8" min="1" max="15" value="${rule?.['traffic-class']?.arp || 8}" required>
                                    </div>
                                </div>
                                <div class="col-md-3">
                                    <div class="mb-3">
                                        <label class="form-label">PDB (ms)</label>
                                        <input type="number" class="form-control tc-pdb" 
                                               placeholder="100" min="0" value="${rule?.['traffic-class']?.pdb || 100}" required>
                                    </div>
                                </div>
                                <div class="col-md-2">
                                    <div class="mb-3">
                                        <label class="form-label">PELR</label>
                                        <input type="number" class="form-control tc-pelr" 
                                               placeholder="6" min="1" max="8" value="${rule?.['traffic-class']?.pelr || 6}" required>
                                    </div>
                                </div>
                            </div>
                            <div class="row">
                                <div class="col-12">
                                    <div class="form-text">
                                        <strong>QCI:</strong> QoS Class Identifier (1-9) | 
                                        <strong>ARP:</strong> Allocation Retention Priority (1-15) | 
                                        <strong>PDB:</strong> Packet Delay Budget | 
                                        <strong>PELR:</strong> Packet Error Loss Rate (1-8)
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        container.insertAdjacentHTML('beforeend', ruleHtml);
    }

    // UPF Management Methods
    collectUpfData() {
        const upfs = {};
        const upfEntries = document.querySelectorAll('.upf-entry');
        
        upfEntries.forEach(entry => {
            const name = entry.querySelector('.upf-name')?.value?.trim();
            const port = entry.querySelector('.upf-port')?.value;
            
            if (name) {
                upfs[name] = {
                    'upf-port': port ? parseInt(port) : undefined
                };
            }
        });
        
        return upfs;
    }

    loadUpfData(upfs) {
        const container = document.getElementById('upf-container');
        if (!container) return;
        
        container.innerHTML = '';
        
        const upfEntries = Object.entries(upfs);
        if (upfEntries.length === 0) {
            upfEntries.push(['', {}]); // Add empty entry
        }
        
        upfEntries.forEach(([upfName, upfConfig], index) => {
            const upfHtml = `
                <div class="upf-entry row mb-3">
                    <div class="col-md-8">
                        <div class="mb-3">
                            <label class="form-label">UPF Name</label>
                            <input type="text" class="form-control upf-name" 
                                   placeholder="e.g., upf-${index + 1}.example.com" value="${upfName || ''}">
                        </div>
                    </div>
                    <div class="col-md-2">
                        <div class="mb-3">
                            <label class="form-label">UPF Port</label>
                            <input type="number" class="form-control upf-port" 
                                   placeholder="8805" min="1" max="65535" value="${upfConfig['upf-port'] || ''}">
                        </div>
                    </div>
                    <div class="col-md-2 d-flex align-items-end">
                        <button type="button" class="btn btn-outline-danger mb-3" onclick="removeUpf(this)">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </div>
            `;
            container.insertAdjacentHTML('beforeend', upfHtml);
        });
    }
}

// Global helper functions for UI interactions
window.addGnb = function() {
    const container = document.getElementById('gnb-container');
    if (!container) return;
    
    const gnbCount = container.querySelectorAll('.gnb-entry').length;
    const gnbHtml = `
        <div class="gnb-entry row mb-3">
            <div class="col-md-5">
                <div class="mb-3">
                    <label class="form-label">gNodeB Name</label>
                    <input type="text" class="form-control gnb-name" 
                           placeholder="e.g., gnb-${gnbCount + 1}" required>
                </div>
            </div>
            <div class="col-md-5">
                <div class="mb-3">
                    <label class="form-label">gNodeB TAC</label>
                    <input type="number" class="form-control gnb-tac" 
                           placeholder="e.g., ${gnbCount + 1}" min="1" max="16777215" required>
                </div>
            </div>
            <div class="col-md-2 d-flex align-items-end">
                <button type="button" class="btn btn-outline-danger mb-3" onclick="removeGnb(this)">
                    <i class="fas fa-trash"></i>
                </button>
            </div>
        </div>
    `;
    container.insertAdjacentHTML('beforeend', gnbHtml);
};

window.addUpf = function() {
    const container = document.getElementById('upf-container');
    if (!container) return;
    
    const upfCount = container.querySelectorAll('.upf-entry').length;
    const upfHtml = `
        <div class="upf-entry row mb-3">
            <div class="col-md-8">
                <div class="mb-3">
                    <label class="form-label">UPF Name</label>
                    <input type="text" class="form-control upf-name" 
                           placeholder="e.g., upf-${upfCount + 1}.example.com">
                </div>
            </div>
            <div class="col-md-2">
                <div class="mb-3">
                    <label class="form-label">UPF Port</label>
                    <input type="number" class="form-control upf-port" 
                           placeholder="8805" min="1" max="65535">
                </div>
            </div>
            <div class="col-md-2 d-flex align-items-end">
                <button type="button" class="btn btn-outline-danger mb-3" onclick="removeUpf(this)">
                    <i class="fas fa-trash"></i>
                </button>
            </div>
        </div>
    `;
    container.insertAdjacentHTML('beforeend', upfHtml);
};

window.removeUpf = function(button) {
    const upfEntry = button.closest('.upf-entry');
    const container = document.getElementById('upf-container');
    
    // Don't allow removing if it's the last one
    if (container.querySelectorAll('.upf-entry').length > 1) {
        upfEntry.remove();
    } else {
        // Clear the fields instead of removing the entry
        upfEntry.querySelector('.upf-name').value = '';
        upfEntry.querySelector('.upf-port').value = '';
    }
};

window.removeGnb = function(button) {
    const gnbEntry = button.closest('.gnb-entry');
    const container = document.getElementById('gnb-container');
    
    // Don't allow removing if it's the last one
    if (container.querySelectorAll('.gnb-entry').length > 1) {
        gnbEntry.remove();
    } else {
        alert('At least one gNodeB is required');
    }
};

window.addApplicationRule = function() {
    const container = document.getElementById('app-rules-container');
    if (!container) return;
    
    const ruleCount = container.querySelectorAll('.app-rule-entry').length;
    
    // Create a temporary NetworkSliceManager instance to use the method
    const tempManager = new NetworkSliceManager();
    tempManager.addApplicationRuleEntry(null, ruleCount);
};

window.removeApplicationRule = function(button) {
    const ruleEntry = button.closest('.app-rule-entry');
    ruleEntry.remove();
};

window.showNetworkSliceDetails = function(sliceName) {
    // This should be handled by the main application
    if (window.app && window.app.networkSliceManager) {
        window.app.networkSliceManager.showDetails(sliceName);
    } else {
        console.warn('Network slice manager not available');
    }
};
