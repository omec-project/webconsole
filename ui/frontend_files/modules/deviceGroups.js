// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

import { BaseManager } from './baseManager.js';
import { API_BASE } from '../app.js';

export class DeviceGroupManager extends BaseManager {
    constructor() {
        super('/device-group', 'device-groups-list');
        this.type = 'device-group';
        this.displayName = 'Device Group';
    }

    // Override loadData to fetch complete device group details
    async loadData() {
        try {
            this.showLoading();
            
            // First, get the list of device group names
            const response = await fetch(`${API_BASE}${this.apiEndpoint}`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const groupNames = await response.json();
            console.log('Device group names:', groupNames);
            
            // Check if we got valid data
            if (!Array.isArray(groupNames)) {
                console.error('Expected array of group names, got:', groupNames);
                this.showError('Invalid response format from server');
                return;
            }
            
            // If no groups, show empty state
            if (groupNames.length === 0) {
                this.data = [];
                this.render([]);
                return;
            }
            
            // Then, fetch complete details for each group
            const groupDetails = [];
            for (const groupName of groupNames) {
                try {
                    if (typeof groupName !== 'string') {
                        console.warn('Invalid group name:', groupName);
                        continue;
                    }
                    
                    const detailResponse = await fetch(`${API_BASE}${this.apiEndpoint}/${encodeURIComponent(groupName)}`);
                    if (detailResponse.ok) {
                        const groupDetail = await detailResponse.json();
                        groupDetails.push(groupDetail);
                    } else {
                        console.warn(`Failed to load details for group ${groupName}: ${detailResponse.status}`);
                    }
                } catch (error) {
                    console.error(`Failed to load details for group ${groupName}:`, error);
                }
            }
            
            console.log('Complete device group details:', groupDetails);
            
            this.data = groupDetails;
            this.render(groupDetails);
            
        } catch (error) {
            this.showError(`Failed to load device groups: ${error.message}`);
            console.error('Load device groups error:', error);
        }
    }

    render(groups) {
        const container = document.getElementById(this.containerId);
        
        if (!container) {
            console.error('Container element not found:', this.containerId);
            return;
        }
        
        if (!groups || !Array.isArray(groups) || groups.length === 0) {
            this.showEmpty('No device groups found');
            return;
        }
        
        let html = '<div class="table-responsive"><table class="table table-striped">';
        html += '<thead><tr><th>Group Name</th><th>IMSIs</th><th>Site Info</th><th>IP Domain</th><th>Actions</th></tr></thead><tbody>';
        
        groups.forEach(group => {
            // Safely extract properties with fallbacks
            const groupName = (group && group['group-name']) || 'N/A';
            const imsis = (group && Array.isArray(group.imsis)) ? group.imsis : [];
            const siteInfo = (group && group['site-info']) || 'N/A';
            const ipDomainName = (group && group['ip-domain-name']) || 'N/A';
            
            html += `
                <tr class="device-group-row" onclick="showDeviceGroupDetails('${groupName}')" style="cursor: pointer;">
                    <td><strong>${groupName}</strong></td>
                    <td>
                        <span class="badge bg-secondary">${imsis.length} IMSIs</span>
                        ${imsis.length > 0 ? `<br><small class="text-muted">${imsis.slice(0, 3).join(', ')}${imsis.length > 3 ? '...' : ''}</small>` : ''}
                    </td>
                    <td>${siteInfo}</td>
                    <td>${ipDomainName}</td>
                    <td onclick="event.stopPropagation();">
                        <button class="btn btn-sm btn-outline-primary me-1" 
                                onclick="editItem('${this.type}', '${groupName}')">
                            <i class="fas fa-edit"></i> Edit
                        </button>
                        <button class="btn btn-sm btn-outline-danger" 
                                onclick="deleteItem('${this.type}', '${groupName}')">
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
                <label class="form-label">Group Name</label>
                <input type="text" class="form-control" id="group_name" 
                       ${isEdit ? 'readonly' : ''} required>
            </div>
            
            <h6 class="mt-4 mb-3">IMSI Configuration</h6>
            <div class="mb-3">
                <label class="form-label">IMSIs</label>
                <textarea class="form-control" id="imsis" rows="4" 
                          placeholder="Enter IMSIs, one per line&#10;e.g.:&#10;001010000000001&#10;001010000000002&#10;001010000000003"></textarea>
                <div class="form-text">Enter one IMSI per line (15 digits each)</div>
            </div>
            
            <h6 class="mt-4 mb-3">Site Information</h6>
            <div class="mb-3">
                <label class="form-label">Site Info</label>
                <input type="text" class="form-control" id="site_info" 
                       placeholder="e.g., site-1">
            </div>
            
            <h6 class="mt-4 mb-3">IP Domain Configuration</h6>
            <div class="mb-3">
                <label class="form-label">IP Domain Name</label>
                <input type="text" class="form-control" id="ip_domain_name" 
                       placeholder="e.g., pool1">
            </div>
            
            <h6 class="mt-4 mb-3">IP Domain Expanded (APN Configuration)</h6>
            <div class="mb-3">
                <label class="form-label">DNN (Data Network Name)</label>
                <input type="text" class="form-control" id="dnn" 
                       placeholder="e.g., internet">
            </div>
            
            <div class="row">
                <div class="col-md-6">
                    <div class="mb-3">
                        <label class="form-label">UE IP Pool</label>
                        <input type="text" class="form-control" id="ue_ip_pool" 
                               placeholder="e.g., 172.250.0.0/16">
                    </div>
                </div>
                <div class="col-md-6">
                    <div class="mb-3">
                        <label class="form-label">MTU</label>
                        <input type="number" class="form-control" id="mtu" 
                               placeholder="e.g., 1460" min="1200" max="9000">
                    </div>
                </div>
            </div>
            
            <div class="row">
                <div class="col-md-6">
                    <div class="mb-3">
                        <label class="form-label">Primary DNS</label>
                        <input type="text" class="form-control" id="dns_primary" 
                               placeholder="e.g., 8.8.8.8">
                    </div>
                </div>
                <div class="col-md-6">
                    <div class="mb-3">
                        <label class="form-label">Secondary DNS</label>
                        <input type="text" class="form-control" id="dns_secondary" 
                               placeholder="e.g., 8.8.4.4">
                    </div>
                </div>
            </div>
            
            <h6 class="mt-4 mb-3">QoS Configuration</h6>
            <div class="row">
                <div class="col-md-4">
                    <div class="mb-3">
                        <label class="form-label">Uplink MBR</label>
                        <input type="number" class="form-control" id="dnn_mbr_uplink" 
                               placeholder="e.g., 100" min="0">
                        <div class="form-text">Mbps</div>
                    </div>
                </div>
                <div class="col-md-4">
                    <div class="mb-3">
                        <label class="form-label">Downlink MBR</label>
                        <input type="number" class="form-control" id="dnn_mbr_downlink" 
                               placeholder="e.g., 200" min="0">
                        <div class="form-text">Mbps</div>
                    </div>
                </div>
                <div class="col-md-4">
                    <div class="mb-3">
                        <label class="form-label">Bitrate Unit</label>
                        <select class="form-select" id="bitrate_unit">
                            <option value="Mbps">Mbps</option>
                            <option value="Kbps">Kbps</option>
                            <option value="Gbps">Gbps</option>
                        </select>
                    </div>
                </div>
            </div>
            <h6 class="mt-4 mb-3">Traffic Class Info</h6>
            <div class="row">
                <div class="col-md-4">
                    <div class="mb-3">
                        <label class="form-label">Traffic Class Name</label>
                        <input type="text" class="form-control" id="traffic_class_name" placeholder="e.g., default">
                    </div>
                </div>
                <div class="col-md-2">
                    <div class="mb-3">
                        <label class="form-label">QCI/5QI/QFI</label>
                        <input type="number" class="form-control" id="traffic_class_qci" min="0" placeholder="e.g., 9">
                    </div>
                </div>
                <div class="col-md-2">
                    <div class="mb-3">
                        <label class="form-label">ARP (Priority)</label>
                        <input type="number" class="form-control" id="traffic_class_arp" min="0" placeholder="e.g., 1">
                    </div>
                </div>
                <div class="col-md-2">
                    <div class="mb-3">
                        <label class="form-label">PDB (ms)</label>
                        <input type="number" class="form-control" id="traffic_class_pdb" min="0" placeholder="e.g., 300">
                    </div>
                </div>
                <div class="col-md-2">
                    <div class="mb-3">
                        <label class="form-label">PELR (%)</label>
                        <input type="number" class="form-control" id="traffic_class_pelr" min="0" max="100" placeholder="e.g., 1">
                    </div>
                </div>
            </div>
        `;
    }


    validateFormData(data) {
        const errors = [];
        
        if (!data.group_name || String(data.group_name).trim() === '') {
            errors.push('Group name is required');
        }
        
        // Validate IMSIs
        if (data.imsis && String(data.imsis).trim() !== '') {
            const imsiList = String(data.imsis).split('\n').map(imsi => imsi.trim()).filter(imsi => imsi);
            for (const imsi of imsiList) {
                if (!/^\d{15}$/.test(imsi)) {
                    errors.push(`Invalid IMSI format: ${imsi}. IMSIs must be exactly 15 digits`);
                    break;
                }
            }
        }
        
        // Validate IP Pool format if provided
        if (data.ue_ip_pool && String(data.ue_ip_pool).trim() !== '') {
            const ipPoolRegex = /^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/;
            if (!ipPoolRegex.test(String(data.ue_ip_pool))) {
                errors.push('UE IP Pool must be in CIDR format (e.g., 172.250.0.0/16)');
            }
        }
        
        // Validate DNS IPs if provided
        const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/;
        if (data.dns_primary && String(data.dns_primary).trim() !== '' && !ipRegex.test(String(data.dns_primary))) {
            errors.push('Primary DNS must be a valid IP address');
        }
        
        if (data.dns_secondary && String(data.dns_secondary).trim() !== '' && !ipRegex.test(String(data.dns_secondary))) {
            errors.push('Secondary DNS must be a valid IP address');
        }
        
        // Validate MTU range
        if (data.mtu) {
            const mtuNum = parseInt(data.mtu);
            if (isNaN(mtuNum) || mtuNum < 1200 || mtuNum > 9000) {
                errors.push('MTU must be a number between 1200 and 9000');
            }
        }
        
        return {
            isValid: errors.length === 0,
            errors: errors
        };
    }

    preparePayload(formData, isEdit = false) {
        // Process IMSIs from textarea
        const imsisList = [];
        if (formData.imsis && formData.imsis.trim() !== '') {
            imsisList.push(...formData.imsis.split('\n').map(imsi => imsi.trim()).filter(imsi => imsi));
        }

        // Prepare IP Domain Expanded structure
        const ipDomainExpanded = {};
        
        if (formData.dnn) ipDomainExpanded.dnn = formData.dnn;
        if (formData.ue_ip_pool) ipDomainExpanded['ue-ip-pool'] = formData.ue_ip_pool;
        if (formData.dns_primary) ipDomainExpanded['dns-primary'] = formData.dns_primary;
        if (formData.dns_secondary) ipDomainExpanded['dns-secondary'] = formData.dns_secondary;
        if (formData.mtu) ipDomainExpanded.mtu = parseInt(formData.mtu);

        // Prepare UE DNN QoS if any values are provided
        const ueDnnQos = {};
        if (formData.dnn_mbr_uplink) ueDnnQos['dnn-mbr-uplink'] = parseInt(formData.dnn_mbr_uplink);
        if (formData.dnn_mbr_downlink) ueDnnQos['dnn-mbr-downlink'] = parseInt(formData.dnn_mbr_downlink);
        if (formData.bitrate_unit) ueDnnQos['bitrate-unit'] = formData.bitrate_unit;

        // Prepare TrafficClassInfo if any values are provided
        const trafficClassInfo = {};
        if (formData.traffic_class_name) trafficClassInfo['name'] = formData.traffic_class_name;
        if (formData.traffic_class_qci) trafficClassInfo['qci'] = parseInt(formData.traffic_class_qci);
        if (formData.traffic_class_arp) trafficClassInfo['arp'] = parseInt(formData.traffic_class_arp);
        if (formData.traffic_class_pdb) trafficClassInfo['pdb'] = parseInt(formData.traffic_class_pdb);
        if (formData.traffic_class_pelr) trafficClassInfo['pelr'] = parseInt(formData.traffic_class_pelr);

        if (Object.keys(trafficClassInfo).length > 0) {
            ueDnnQos['traffic-class'] = trafficClassInfo;
        }
        if (Object.keys(ueDnnQos).length > 0) {
            ipDomainExpanded['ue-dnn-qos'] = ueDnnQos;
        }

        const payload = {
            "group-name": formData.group_name,
            "imsis": imsisList,
            "site-info": formData.site_info,
            "ip-domain-name": formData.ip_domain_name,
            "ip-domain-expanded": ipDomainExpanded
        };

        return payload;
    }

    // Override createItem to include group name in URL for device groups
    async createItem(itemData) {
        try {
            const groupName = itemData['group-name'];
            const response = await fetch(`${API_BASE}${this.apiEndpoint}/${groupName}`, {
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

    async loadItemData(name) {
        try {
            const response = await fetch(`${API_BASE}${this.apiEndpoint}/${encodeURIComponent(name)}`);
            if (response.ok) {
                const data = await response.json();
                
                // Populate basic fields
                this.setFieldValue('group_name', data['group-name']);
                this.setFieldValue('site_info', data['site-info']);
                this.setFieldValue('ip_domain_name', data['ip-domain-name']);
                
                // Populate IMSIs (convert array to textarea)
                if (data.imsis && data.imsis.length > 0) {
                    this.setFieldValue('imsis', data.imsis.join('\n'));
                }
                
                // Populate IP Domain Expanded fields
                const ipDomainExpanded = data['ip-domain-expanded'] || {};
                this.setFieldValue('dnn', ipDomainExpanded.dnn);
                this.setFieldValue('ue_ip_pool', ipDomainExpanded['ue-ip-pool']);
                this.setFieldValue('dns_primary', ipDomainExpanded['dns-primary']);
                this.setFieldValue('dns_secondary', ipDomainExpanded['dns-secondary']);
                this.setFieldValue('mtu', ipDomainExpanded.mtu);
                
                // Populate UE DNN QoS fields
                const ueDnnQos = ipDomainExpanded['ue-dnn-qos'] || {};
                this.setFieldValue('dnn_mbr_uplink', ueDnnQos['dnn-mbr-uplink']);
                this.setFieldValue('dnn_mbr_downlink', ueDnnQos['dnn-mbr-downlink']);
                this.setFieldValue('bitrate_unit', ueDnnQos['bitrate-unit'] || 'Mbps');
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

    // New methods for details view
    async showDetails(groupName) {
        try {
            const response = await fetch(`${API_BASE}${this.apiEndpoint}/${encodeURIComponent(groupName)}`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const groupData = await response.json();
            this.currentGroupData = groupData;
            this.currentGroupName = groupName;
            this.renderDetailsView(groupData);
            
        } catch (error) {
            console.error('Failed to load device group details:', error);
            // Show error notification
            window.app?.notificationManager?.showNotification('Error loading device group details', 'error');
        }
    }

    renderDetailsView(groupData) {
        const container = document.getElementById('device-group-details-content');
        const title = document.getElementById('device-group-detail-title');
        
        if (!container || !title) {
            console.error('Details container not found');
            return;
        }

        const groupName = groupData['group-name'] || 'Unknown';
        title.textContent = `Device Group: ${groupName}`;

        const ipDomainExpanded = groupData['ip-domain-expanded'] || {};
        const ueDnnQos = ipDomainExpanded['ue-dnn-qos'] || {};

        const html = `
            <div id="details-view-mode">
                ${this.renderReadOnlyDetails(groupData)}
            </div>
            <div id="details-edit-mode" style="display: none;">
                ${this.renderEditableDetails(groupData)}
            </div>
        `;

        container.innerHTML = html;
    }

    renderReadOnlyDetails(groupData) {
        const ipDomainExpanded = groupData['ip-domain-expanded'] || {};
        const ueDnnQos = ipDomainExpanded['ue-dnn-qos'] || {};
        const imsis = groupData.imsis || [];

        return `
            <div class="row">
                <div class="col-md-6">
                    <div class="card mb-3">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-info-circle me-2"></i>Basic Information</h6>
                        </div>
                        <div class="card-body">
                            <div class="mb-2">
                                <strong>Group Name:</strong> ${groupData['group-name'] || 'N/A'}
                            </div>
                            <div class="mb-2">
                                <strong>Site Info:</strong> ${groupData['site-info'] || 'N/A'}
                            </div>
                            <div class="mb-2">
                                <strong>IP Domain Name:</strong> ${groupData['ip-domain-name'] || 'N/A'}
                            </div>
                        </div>
                    </div>

                    <div class="card mb-3">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-sim-card me-2"></i>IMSI Configuration</h6>
                        </div>
                        <div class="card-body">
                            <div class="mb-2">
                                <strong>Total IMSIs:</strong> <span class="badge bg-primary">${imsis.length}</span>
                            </div>
                            ${imsis.length > 0 ? `
                                <div class="mb-2">
                                    <strong>IMSIs:</strong>
                                    <div class="mt-2" style="max-height: 200px; overflow-y: auto;">
                                        ${imsis.map(imsi => `<div class="badge bg-light text-dark me-1 mb-1">${imsi}</div>`).join('')}
                                    </div>
                                </div>
                            ` : '<p class="text-muted">No IMSIs configured</p>'}
                        </div>
                    </div>
                </div>

                <div class="col-md-6">
                    <div class="card mb-3">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-network-wired me-2"></i>IP Domain Configuration</h6>
                        </div>
                        <div class="card-body">
                            <div class="mb-2">
                                <strong>DNN:</strong> ${ipDomainExpanded.dnn || 'N/A'}
                            </div>
                            <div class="mb-2">
                                <strong>UE IP Pool:</strong> ${ipDomainExpanded['ue-ip-pool'] || 'N/A'}
                            </div>
                            <div class="mb-2">
                                <strong>MTU:</strong> ${ipDomainExpanded.mtu || 'N/A'}
                            </div>
                            <div class="mb-2">
                                <strong>Primary DNS:</strong> ${ipDomainExpanded['dns-primary'] || 'N/A'}
                            </div>
                            <div class="mb-2">
                                <strong>Secondary DNS:</strong> ${ipDomainExpanded['dns-secondary'] || 'N/A'}
                            </div>
                        </div>
                    </div>

                    <div class="card mb-3">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-tachometer-alt me-2"></i>QoS Configuration</h6>
                        </div>
                        <div class="card-body">
                            <div class="mb-2">
                                <strong>Uplink MBR:</strong> ${ueDnnQos['dnn-mbr-uplink'] || 'N/A'} ${ueDnnQos['bitrate-unit'] || ''}
                            </div>
                            <div class="mb-2">
                                <strong>Downlink MBR:</strong> ${ueDnnQos['dnn-mbr-downlink'] || 'N/A'} ${ueDnnQos['bitrate-unit'] || ''}
                            </div>
                            <div class="mb-2">
                                <strong>Bitrate Unit:</strong> ${ueDnnQos['bitrate-unit'] || 'N/A'}
                            </div>
                            ${ueDnnQos['traffic-class'] ? `
                            <hr>
                            <h6 class="mb-3">Traffic Class Info</h6>
                            <div class="mb-2">
                                <strong>Name:</strong> ${ueDnnQos['traffic-class'].name || 'N/A'}
                            </div>
                            <div class="mb-2">
                                <strong>QCI/5QI/QFI:</strong> ${ueDnnQos['traffic-class'].qci || 'N/A'}
                            </div>
                            <div class="mb-2">
                                <strong>ARP (Priority):</strong> ${ueDnnQos['traffic-class'].arp || 'N/A'}
                            </div>
                            <div class="mb-2">
                                <strong>PDB (ms):</strong> ${ueDnnQos['traffic-class'].pdb || 'N/A'}
                            </div>
                            <div class="mb-2">
                                <strong>PELR (%):</strong> ${ueDnnQos['traffic-class'].pelr || 'N/A'}
                            </div>
                            ` : ''}
                        </div>
                    </div>
                    
                </div>
            </div>
        `;
    }

    renderEditableDetails(groupData) {
        const ipDomainExpanded = groupData['ip-domain-expanded'] || {};
        const ueDnnQos = ipDomainExpanded['ue-dnn-qos'] || {};
        const imsis = groupData.imsis || [];

        return `
            <form id="detailsEditForm">
                <div class="row">
                    <div class="col-md-6">
                        <div class="card mb-3">
                            <div class="card-header">
                                <h6 class="mb-0"><i class="fas fa-info-circle me-2"></i>Basic Information</h6>
                            </div>
                            <div class="card-body">
                                <div class="mb-3">
                                    <label class="form-label">Group Name</label>
                                    <input type="text" class="form-control" id="edit_group_name" 
                                           value="${groupData['group-name'] || ''}" readonly>
                                </div>
                                <div class="mb-3">
                                    <label class="form-label">Site Info</label>
                                    <input type="text" class="form-control" id="edit_site_info" 
                                           value="${groupData['site-info'] || ''}" placeholder="e.g., site-1">
                                </div>
                                <div class="mb-3">
                                    <label class="form-label">IP Domain Name</label>
                                    <input type="text" class="form-control" id="edit_ip_domain_name" 
                                           value="${groupData['ip-domain-name'] || ''}" placeholder="e.g., pool1">
                                </div>
                            </div>
                        </div>

                        <div class="card mb-3">
                            <div class="card-header">
                                <h6 class="mb-0"><i class="fas fa-sim-card me-2"></i>IMSI Configuration</h6>
                            </div>
                            <div class="card-body">
                                <div class="mb-3">
                                    <label class="form-label">IMSIs</label>
                                    <textarea class="form-control" id="edit_imsis" rows="6" 
                                              placeholder="Enter IMSIs, one per line">${imsis.join('\n')}</textarea>
                                    <div class="form-text">Enter one IMSI per line (15 digits each)</div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="col-md-6">
                        <div class="card mb-3">
                            <div class="card-header">
                                <h6 class="mb-0"><i class="fas fa-network-wired me-2"></i>IP Domain Configuration</h6>
                            </div>
                            <div class="card-body">
                                <div class="mb-3">
                                    <label class="form-label">DNN (Data Network Name)</label>
                                    <input type="text" class="form-control" id="edit_dnn" 
                                           value="${ipDomainExpanded.dnn || ''}" placeholder="e.g., internet">
                                </div>
                                <div class="mb-3">
                                    <label class="form-label">UE IP Pool</label>
                                    <input type="text" class="form-control" id="edit_ue_ip_pool" 
                                           value="${ipDomainExpanded['ue-ip-pool'] || ''}" placeholder="e.g., 172.250.0.0/16">
                                </div>
                                <div class="mb-3">
                                    <label class="form-label">MTU</label>
                                    <input type="number" class="form-control" id="edit_mtu" 
                                           value="${ipDomainExpanded.mtu || ''}" placeholder="e.g., 1460" min="1200" max="9000">
                                </div>
                                <div class="row">
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">Primary DNS</label>
                                            <input type="text" class="form-control" id="edit_dns_primary" 
                                                   value="${ipDomainExpanded['dns-primary'] || ''}" placeholder="e.g., 8.8.8.8">
                                        </div>
                                    </div>
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">Secondary DNS</label>
                                            <input type="text" class="form-control" id="edit_dns_secondary" 
                                                   value="${ipDomainExpanded['dns-secondary'] || ''}" placeholder="e.g., 8.8.4.4">
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="card mb-3">
                            <div class="card-header">
                                <h6 class="mb-0"><i class="fas fa-tachometer-alt me-2"></i>QoS Configuration</h6>
                            </div>
                            <div class="card-body">
                                <div class="row">
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">Uplink MBR</label>
                                            <input type="number" class="form-control" id="edit_dnn_mbr_uplink" 
                                                   value="${ueDnnQos['dnn-mbr-uplink'] || ''}" placeholder="e.g., 100" min="0">
                                            <div class="form-text">Mbps</div>
                                        </div>
                                    </div>
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">Downlink MBR</label>
                                            <input type="number" class="form-control" id="edit_dnn_mbr_downlink" 
                                                   value="${ueDnnQos['dnn-mbr-downlink'] || ''}" placeholder="e.g., 200" min="0">
                                            <div class="form-text">Mbps</div>
                                        </div>
                                    </div>
                                </div>
                                <div class="mb-3">
                                    <label class="form-label">Bitrate Unit</label>
                                    <select class="form-select" id="edit_bitrate_unit">
                                        <option value="Mbps" ${ueDnnQos['bitrate-unit'] === 'Mbps' ? 'selected' : ''}>Mbps</option>
                                        <option value="Kbps" ${ueDnnQos['bitrate-unit'] === 'Kbps' ? 'selected' : ''}>Kbps</option>
                                        <option value="Gbps" ${ueDnnQos['bitrate-unit'] === 'Gbps' ? 'selected' : ''}>Gbps</option>
                                    </select>
                                </div>

                                <h6 class="mt-4 mb-3">Traffic Class Info</h6>
                                <div class="row">
                                    <div class="col-md-4">
                                        <div class="mb-3">
                                            <label class="form-label">Traffic Class Name</label>
                                            <input type="text" class="form-control" id="edit_traffic_class_name" 
                                                   value="${(ueDnnQos['traffic-class'] && ueDnnQos['traffic-class'].name) || ''}" placeholder="e.g., default">
                                        </div>
                                    </div>
                                    <div class="col-md-2">
                                        <div class="mb-3">
                                            <label class="form-label">QCI/5QI/QFI</label>
                                            <input type="number" class="form-control" id="edit_traffic_class_qci" 
                                                   value="${(ueDnnQos['traffic-class'] && ueDnnQos['traffic-class'].qci) || ''}" min="0" placeholder="e.g., 9">
                                        </div>
                                    </div>
                                    <div class="col-md-2">
                                        <div class="mb-3">
                                            <label class="form-label">ARP (Priority)</label>
                                            <input type="number" class="form-control" id="edit_traffic_class_arp" 
                                                   value="${(ueDnnQos['traffic-class'] && ueDnnQos['traffic-class'].arp) || ''}" min="0" placeholder="e.g., 1">
                                        </div>
                                    </div>
                                    <div class="col-md-2">
                                        <div class="mb-3">
                                            <label class="form-label">PDB (ms)</label>
                                            <input type="number" class="form-control" id="edit_traffic_class_pdb" 
                                                   value="${(ueDnnQos['traffic-class'] && ueDnnQos['traffic-class'].pdb) || ''}" min="0" placeholder="e.g., 300">
                                        </div>
                                    </div>
                                    <div class="col-md-2">
                                        <div class="mb-3">
                                            <label class="form-label">PELR (%)</label>
                                            <input type="number" class="form-control" id="edit_traffic_class_pelr" 
                                                   value="${(ueDnnQos['traffic-class'] && ueDnnQos['traffic-class'].pelr) || ''}" min="0" max="100" placeholder="e.g., 1">
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                
                <div class="row">
                    <div class="col-12">
                        <div class="d-flex justify-content-end">
                            <button type="button" class="btn btn-secondary me-2" onclick="cancelEdit()">Cancel</button>
                            <button type="button" class="btn btn-primary" onclick="saveDetailsEdit()">Save Changes</button>
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
            await this.updateItem(this.currentGroupName, payload);
            
            // Refresh the details view
            await this.showDetails(this.currentGroupName);
            this.toggleEditMode(false);
            
            window.app?.notificationManager?.showNotification('Device group updated successfully!', 'success');
            
        } catch (error) {
            console.error('Failed to save device group:', error);
            window.app?.notificationManager?.showNotification(`Failed to save device group: ${error.message}`, 'error');
        }
    }

    getEditFormData() {
        return {
            group_name: document.getElementById('edit_group_name')?.value || '',
            site_info: document.getElementById('edit_site_info')?.value || '',
            ip_domain_name: document.getElementById('edit_ip_domain_name')?.value || '',
            imsis: document.getElementById('edit_imsis')?.value || '',
            dnn: document.getElementById('edit_dnn')?.value || '',
            ue_ip_pool: document.getElementById('edit_ue_ip_pool')?.value || '',
            mtu: document.getElementById('edit_mtu')?.value || '',
            dns_primary: document.getElementById('edit_dns_primary')?.value || '',
            dns_secondary: document.getElementById('edit_dns_secondary')?.value || '',
            dnn_mbr_uplink: document.getElementById('edit_dnn_mbr_uplink')?.value || '',
            dnn_mbr_downlink: document.getElementById('edit_dnn_mbr_downlink')?.value || '',
            bitrate_unit: document.getElementById('edit_bitrate_unit')?.value || 'Mbps',
            traffic_class_name: document.getElementById('edit_traffic_class_name')?.value || '',
            traffic_class_qci: document.getElementById('edit_traffic_class_qci')?.value || '',
            traffic_class_arp: document.getElementById('edit_traffic_class_arp')?.value || '',
            traffic_class_pdb: document.getElementById('edit_traffic_class_pdb')?.value || '',
            traffic_class_pelr: document.getElementById('edit_traffic_class_pelr')?.value || ''
        };
    }

    toggleEditMode(enable = null) {
        const detailsView = document.getElementById('details-view-mode');
        const editView = document.getElementById('details-edit-mode');
        const editBtn = document.getElementById('edit-device-group-btn');
        
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
        }
    }

    async deleteFromDetails() {
        try {
            await this.deleteItem(this.currentGroupName);
            window.app?.notificationManager?.showNotification('Device group deleted successfully!', 'success');
            
            // Navigate back to the list
            window.showSection('device-groups');
            
        } catch (error) {
            console.error('Failed to delete device group:', error);
            window.app?.notificationManager?.showNotification(`Failed to delete device group: ${error.message}`, 'error');
        }
    }
}
