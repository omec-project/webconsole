// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

import { BaseManager } from './baseManager.js';

export class UpfManager extends BaseManager {
    constructor() {
        super('/inventory/upf', 'upf-list');
        this.type = 'upf';
        this.displayName = 'UPF';
    }

    render(upfs) {
        const container = document.getElementById(this.containerId);
        
        if (!upfs || upfs.length === 0) {
            this.showEmpty('No UPFs found');
            return;
        }
        
        let html = '<div class="table-responsive"><table class="table table-striped">';
        html += '<thead><tr><th>Hostname</th><th>Port</th><th>Actions</th></tr></thead><tbody>';
        
        upfs.forEach(upf => {
            html += `
                <tr>
                    <td><strong>${upf.hostname || 'N/A'}</strong></td>
                    <td>${upf.port || 'N/A'}</td>
                    <td>
                        <button class="btn btn-sm btn-outline-primary me-1" 
                                onclick="editItem('${this.type}', '${upf.hostname}')">
                            <i class="fas fa-edit"></i> Edit
                        </button>
                        <button class="btn btn-sm btn-outline-danger" 
                                onclick="deleteItem('${this.type}', '${upf.hostname}')">
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
                <label class="form-label">UPF Hostname</label>
                <input type="text" class="form-control" id="hostname" 
                       ${isEdit ? 'readonly' : ''} required>
            </div>
            <div class="mb-3">
                <label class="form-label">Port</label>
                <input type="text" class="form-control" id="port" 
                       placeholder="e.g., 8805" required>
                <div class="form-text">Port number as string</div>
            </div>
        `;
    }

    validateFormData(data) {
        const errors = [];
        
        if (!data.hostname || data.hostname.trim() === '') {
            errors.push('UPF hostname is required');
        }
        
        if (!data.port || data.port.trim() === '') {
            errors.push('Port is required');
        }
        
        return {
            isValid: errors.length === 0,
            errors: errors
        };
    }

    preparePayload(formData, isEdit = false) {
        return {
            "hostname": formData.hostname,
            "port": formData.port
        };
    }
}
