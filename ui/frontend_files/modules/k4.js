// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

import { BaseManager } from './baseManager.js';
import { SUBSCRIBER_API_BASE } from '../app.js';

// --- GESTOR PARA LAS CLAVES K4 ---
export class K4Manager extends BaseManager {
    constructor() {
        // se usa el BaseManager modificado pasándole el endpoint y la URL base.
        super('/k4opt', 'k4-keys-list', SUBSCRIBER_API_BASE);
        this.type = 'k4-key';
        this.displayName = 'K4 Key';
    }

    render(keys) {
        const container = document.getElementById(this.containerId);
        if (!keys || keys.length === 0) {
            this.showEmpty('No K4 keys found. Add one to provision a subscriber.');
            return;
        }

        let html = '<div class="table-responsive"><table class="table table-striped table-hover">';
        html += '<thead><tr><th>Serial Number (SNO)</th><th>Key Label</th><th>Key Type</th><th>K4 Key</th><th>Actions</th></tr></thead><tbody>';

        keys.forEach(key => {
            // Store both k4_sno and key_label for delete operation
            const k4Identifier = JSON.stringify({sno: key.k4_sno, label: key.key_label});
            html += `
                <tr class="k4-row" onclick="showK4Details('${key.k4_sno}')" style="cursor: pointer;">
                    <td><span class="badge bg-primary fs-6">${key.k4_sno ?? 'N/A'}</span></td>
                    <td><span class="badge bg-info fs-6">${key.key_label ?? 'N/A'}</span></td>
                    <td><span class="badge bg-secondary fs-6">${key.key_type ?? 'N/A'}</span></td>
                    <td><code>${key.k4 && key.k4.trim() !== '' ? key.k4 : 'N/S'}</code></td>
                    <td onclick="event.stopPropagation();">
                        <button class="btn btn-sm btn-outline-primary me-1" title="Edit"
                                onclick="editItem('${this.type}', '${key.k4_sno ?? 'N/A'}')">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn btn-sm btn-outline-danger" title="Delete"
                                onclick="deleteK4Item('${key.k4_sno ?? 'N/A'}', '${key.key_label ?? ''}')">
                            <i class="fas fa-trash"></i>
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
                <label class="form-label">K4 Serial Number (SNO)</label>
                <input type="number" class="form-control" id="k4_sno" min="0" max="255"
                       ${isEdit ? 'readonly disabled' : ''} ${isEdit ? '' : 'required'}>
                <div class="form-text">Value between 0-255 (byte)</div>
            </div>
            <div class="mb-3">
                <label class="form-label">Key Label</label>
                <select class="form-select" id="key_label" ${isEdit ? 'disabled' : ''} ${isEdit ? '' : 'required'}>
                    <option value="">Select key label...</option>
                    <option value="K4_AES">K4_AES</option>
                    <option value="K4_DES">K4_DES</option>
                    <option value="K4_DES3">K4_DES3</option>
                </select>
                ${isEdit ? '<input type="hidden" id="key_label_hidden" />' : ''}
                <div class="form-text">${isEdit ? 'Key Label cannot be changed in edit mode' : 'Select the encryption key label'}</div>
            </div>
            <div class="mb-3">
                <label class="form-label">Key Type</label>
                <select class="form-select" id="key_type" ${isEdit ? 'disabled' : ''} ${isEdit ? '' : 'required'}>
                    <option value="">Select key type...</option>
                    <option value="AES">AES</option>
                    <option value="DES">DES</option>
                    <option value="DES3">DES3</option>
                </select>
                ${isEdit ? '<input type="hidden" id="key_type_hidden" />' : ''}
                <div class="form-text">${isEdit ? 'Key Type cannot be changed in edit mode' : 'Select the encryption algorithm type'}</div>
            </div>
            <div class="mb-3">
                <label class="form-label">K4 Key</label>
                <input type="text" class="form-control" id="k4" 
                       placeholder="e.g., 00112233445566778899aabbccddeeff" 
                       pattern="[0-9a-fA-F]+" required>
                <div class="form-text">Hexadecimal key value${isEdit ? ' — editable in edit mode only' : ''}</div>
            </div>
        `;
    }

    validateFormData(data, isEdit = false) {
        const errors = [];
        
        if (!isEdit) {
            // For creation, validate all fields
            if (data.k4_sno === undefined || data.k4_sno < 0 || data.k4_sno > 255) {
                errors.push('K4 SNO is required and must be between 0-255.');
            }

            if (!data.key_label || data.key_label === '') {
                errors.push('Key Label is required.');
            }

            if (!data.key_type || data.key_type === '') {
                errors.push('Key Type is required.');
            }
        }

        // Always validate k4 key value
        if (!data.k4 || !/^[0-9a-fA-F]+$/.test(data.k4)) {
            errors.push('K4 Key must contain only hexadecimal characters.');
        }

        return { isValid: errors.length === 0, errors: errors };
    }

    preparePayload(formData) {
        return {
            "k4_sno": parseInt(formData.k4_sno),
            "k4": formData.k4.toLowerCase(),
            "key_label": formData.key_label,
            "key_type": formData.key_type
        };
    }
    
    async showEditForm(name) {
        // Llama explícitamente al método genérico de carga de datos del padre.
        await this.loadItemData(name);
    }

    // New methods for details view
    async showDetails(k4Sno) {
        try {
            const response = await fetch(`${this.apiBase}${this.apiEndpoint}/${encodeURIComponent(k4Sno)}`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const k4Data = await response.json();
            this.currentK4Data = k4Data;
            this.currentK4Sno = k4Sno;
            this.renderDetailsView(k4Data);
            
        } catch (error) {
            console.error('Failed to load K4 key details:', error);
            // Show error notification
            window.app?.notificationManager?.showNotification('Error loading K4 key details', 'error');
        }
    }

    renderDetailsView(k4Data) {
        const container = document.getElementById('k4-details-content');
        const title = document.getElementById('k4-detail-title');
        
        if (!container || !title) {
            console.error('Details container not found');
            return;
        }

        const k4Sno = k4Data.k4_sno || 'Unknown';
        title.textContent = `K4 Key: SNO ${k4Sno}`;

        const html = `
            <div id="k4-details-view-mode">
                ${this.renderReadOnlyDetails(k4Data)}
            </div>
            <div id="k4-details-edit-mode" style="display: none;">
                ${this.renderEditableDetails(k4Data)}
            </div>
        `;

        container.innerHTML = html;
    }

    renderReadOnlyDetails(k4Data) {
        return `
            <div class="row justify-content-center">
                <div class="col-md-8">
                    <div class="card">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-key me-2"></i>K4 Key Information</h6>
                        </div>
                        <div class="card-body">
                            <div class="row">
                                <div class="col-md-6">
                                    <div class="mb-3">
                                        <strong>Serial Number (SNO):</strong> 
                                        <div class="mt-1">
                                            <span class="badge bg-primary fs-6">${k4Data.k4_sno ?? 'N/A'}</span>
                                        </div>
                                    </div>
                                </div>
                                <div class="col-md-6">
                                    <div class="mb-3">
                                        <strong>Key Label:</strong> 
                                        <div class="mt-1">
                                            <span class="badge bg-info fs-6">${k4Data.key_label ?? 'N/A'}</span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            <div class="row">
                                <div class="col-md-6">
                                    <div class="mb-3">
                                        <strong>Key Type:</strong> 
                                        <div class="mt-1">
                                            <span class="badge bg-secondary fs-6">${k4Data.key_type ?? 'N/A'}</span>
                                        </div>
                                    </div>
                                </div>
                                <div class="col-md-6">
                                    <div class="mb-3">
                                        <strong>K4 Key:</strong>
                                        <div class="mt-1">
                                            <code class="text-break">${k4Data.k4 && k4Data.k4 !== '' ? k4Data.k4 : 'N/S'}</code>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            
                            <hr>
                            
                            <div class="row">
                                <div class="col-12">
                                    <h6 class="mb-3"><i class="fas fa-info-circle me-2"></i>Technical Information</h6>
                                    <div class="bg-light p-3 rounded">
                                        <div class="row">
                                            <div class="col-md-6">
                                                <small class="text-muted">Key Type:</small>
                                                <div><strong>K4 Authentication Key</strong></div>
                                            </div>
                                            <div class="col-md-6">
                                                <small class="text-muted">Format:</small>
                                                <div><strong>Hexadecimal Characters</strong></div>
                                            </div>
                                        </div>
                                        <div class="row mt-2">
                                            <div class="col-md-6">
                                                <small class="text-muted">Usage:</small>
                                                <div><strong>Subscriber Authentication</strong></div>
                                            </div>
                                            <div class="col-md-6">
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
                </div>
            </div>
        `;
    }

    renderEditableDetails(k4Data) {
        return `
            <form id="k4KeyDetailsEditForm">
                <div class="row justify-content-center">
                    <div class="col-md-8">
                        <div class="card">
                            <div class="card-header">
                                <h6 class="mb-0"><i class="fas fa-edit me-2"></i>Edit K4 Key Information</h6>
                            </div>
                            <div class="card-body">
                                <div class="row">
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">Serial Number (SNO)</label>
                                            <input type="number" class="form-control" id="edit_k4_sno" 
                                                   value="${k4Data.k4_sno || ''}" readonly min="0" max="255" disabled>
                                            <div class="form-text">SNO cannot be changed</div>
                                        </div>
                                    </div>
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">Key Label</label>
                                            <input type="text" class="form-control" id="edit_key_label" 
                                                   value="${k4Data.key_label || ''}" readonly disabled>
                                            <div class="form-text">Key Label cannot be changed</div>
                                        </div>
                                    </div>
                                </div>
                                <div class="row">
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">Key Type</label>
                                            <input type="text" class="form-control" id="edit_key_type" 
                                                   value="${k4Data.key_type || ''}" readonly disabled>
                                            <div class="form-text">Key Type cannot be changed</div>
                                        </div>
                                    </div>
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">K4 Key <span class="text-danger">*</span></label>
                                            <input type="text" class="form-control" id="edit_k4_key" 
                                                   value="${k4Data.k4 || ''}" placeholder="Hexadecimal characters" 
                                                   pattern="[0-9a-fA-F]+" required>
                                            <div class="form-text">Only the K4 key value can be edited</div>
                                        </div>
                                    </div>
                                </div>
                                
                                <hr>
                                
                                <div class="bg-light p-3 rounded mb-3">
                                    <h6 class="mb-2"><i class="fas fa-info-circle me-2"></i>About K4 Keys</h6>
                                    <p class="mb-1 small">
                                        <i class="fas fa-lock me-1"></i>
                                        <strong>Read-only fields:</strong> SNO, Key Label, and Key Type cannot be modified.
                                    </p>
                                    <p class="mb-0 small">
                                        <i class="fas fa-edit me-1"></i>
                                        <strong>Editable:</strong> Only the K4 key value can be updated.
                                    </p>
                                </div>
                            </div>
                        </div>
                        
                        <div class="d-flex justify-content-end mt-3">
                            <button type="button" class="btn btn-secondary me-2" onclick="cancelK4Edit()">Cancel</button>
                            <button type="button" class="btn btn-primary" onclick="saveK4DetailsEdit()">Save Changes</button>
                        </div>
                    </div>
                </div>
            </form>
        `;
    }

    async saveEdit() {
        try {
            const formData = this.getEditFormData();
            const validation = this.validateFormData(formData, true); // isEdit = true
            
            if (!validation.isValid) {
                window.app?.notificationManager?.showNotification(validation.errors.join('<br>'), 'error');
                return;
            }

            // For edit, only send the k4 value, keep other fields from currentK4Data
            const payload = {
                "k4_sno": parseInt(formData.k4_sno),
                "k4": formData.k4.toLowerCase(),
                "key_label": formData.key_label,
                "key_type": formData.key_type
            };
            
            await this.updateItem(this.currentK4Sno, payload);
            
            // Refresh the details view
            await this.showDetails(this.currentK4Sno);
            this.toggleEditMode(false);
            
            window.app?.notificationManager?.showNotification('K4 key updated successfully!', 'success');
            
        } catch (error) {
            console.error('Failed to save K4 key:', error);
            window.app?.notificationManager?.showNotification(`Failed to save K4 key: ${error.message}`, 'error');
        }
    }

    getEditFormData() {
        return {
            k4_sno: document.getElementById('edit_k4_sno')?.value || '',
            k4: document.getElementById('edit_k4_key')?.value || '',
            key_label: document.getElementById('edit_key_label')?.value || '',
            key_type: document.getElementById('edit_key_type')?.value || ''
        };
    }

    async deleteItem(k4Sno, keyLabel) {
        try {
            // Use the new endpoint format: /k4opt/:idsno/:keylabel
            const response = await fetch(
                `${this.apiBase}${this.apiEndpoint}/${encodeURIComponent(k4Sno)}/${encodeURIComponent(keyLabel)}`,
                { method: 'DELETE' }
            );

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(errorText || `HTTP ${response.status}`);
            }

            return response.status === 204 ? {} : await response.json();
        } catch (error) {
            throw error;
        }
    }

    toggleEditMode(enable = null) {
        const detailsView = document.getElementById('k4-details-view-mode');
        const editView = document.getElementById('k4-details-edit-mode');
        const editBtn = document.getElementById('edit-k4-btn');
        
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
            // Use the currentK4Data to get both sno and key_label
            const k4Sno = this.currentK4Data.k4_sno;
            const keyLabel = this.currentK4Data.key_label;
            
            await this.deleteItem(k4Sno, keyLabel);
            window.app?.notificationManager?.showNotification('K4 key deleted successfully!', 'success');
            
            // Navigate back to the list
            window.showSection('k4-keys');
            
        } catch (error) {
            console.error('Failed to delete K4 key:', error);
            window.app?.notificationManager?.showNotification(`Failed to delete K4 key: ${error.message}`, 'error');
        }
    }
}
