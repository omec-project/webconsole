// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

import { BaseManager } from './baseManager.js';

export class GnbManager extends BaseManager {
    constructor() {
        super('/inventory/gnb', 'gnb-list');
        this.type = 'gnb';
        this.displayName = 'gNB';
    }

    render(gnbs) {
        const container = document.getElementById(this.containerId);
        
        if (!gnbs || gnbs.length === 0) {
            this.showEmpty('No gNBs found');
            return;
        }
        
        let html = '<div class="table-responsive"><table class="table table-striped">';
        html += '<thead><tr><th>Name</th><th>TAC</th><th>Actions</th></tr></thead><tbody>';
        
        gnbs.forEach(gnb => {
            html += `
                <tr class="gnb-row" onclick="showGnbDetails('${gnb.name}')" style="cursor: pointer;">
                    <td><strong>${gnb.name || 'N/A'}</strong></td>
                    <td>${gnb.tac || 'N/A'}</td>
                    <td onclick="event.stopPropagation();">
                        <button class="btn btn-sm btn-outline-primary me-1" 
                                onclick="editItem('${this.type}', '${gnb.name}')">
                            <i class="fas fa-edit"></i> Edit
                        </button>
                        <button class="btn btn-sm btn-outline-danger" 
                                onclick="deleteItem('${this.type}', '${gnb.name}')">
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
                <label class="form-label">gNB Name</label>
                <input type="text" class="form-control" id="name" 
                       ${isEdit ? 'readonly' : ''} required>
            </div>
            <div class="mb-3">
                <label class="form-label">TAC (Tracking Area Code)</label>
                <input type="number" class="form-control" id="tac" 
                       placeholder="e.g., 1" min="1" max="16777215">
                <div class="form-text">Optional: Integer value between 1 and 16777215</div>
            </div>
        `;
    }

    validateFormData(data) {
        const errors = [];
        
        if (!data.name || data.name.trim() === '') {
            errors.push('gNB name is required');
        }
        
        if (data.tac && (data.tac < 1 || data.tac > 16777215)) {
            errors.push('TAC must be between 1 and 16777215');
        }
        
        return {
            isValid: errors.length === 0,
            errors: errors
        };
    }

    preparePayload(formData, isEdit = false) {
        const payload = {
            "name": formData.name
        };

        // Only include tac if it's provided
        if (formData.tac && formData.tac.toString().trim() !== '') {
            payload.tac = parseInt(formData.tac);
        }

        return payload;
    }

    // New methods for details view
    async showDetails(gnbName) {
        try {
            const response = await fetch(`${this.apiBase}${this.apiEndpoint}/${encodeURIComponent(gnbName)}`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const gnbData = await response.json();
            this.currentGnbData = gnbData;
            this.currentGnbName = gnbName;
            this.renderDetailsView(gnbData);
            
        } catch (error) {
            console.error('Failed to load gNB details:', error);
            // Show error notification
            window.app?.notificationManager?.showNotification('Error loading gNB details', 'error');
        }
    }

    renderDetailsView(gnbData) {
        const container = document.getElementById('gnb-details-content');
        const title = document.getElementById('gnb-detail-title');
        
        if (!container || !title) {
            console.error('Details container not found');
            return;
        }

        const gnbName = gnbData.name || 'Unknown';
        title.textContent = `gNB: ${gnbName}`;

        const html = `
            <div id="gnb-details-view-mode">
                ${this.renderReadOnlyDetails(gnbData)}
            </div>
            <div id="gnb-details-edit-mode" style="display: none;">
                ${this.renderEditableDetails(gnbData)}
            </div>
        `;

        container.innerHTML = html;
    }

    renderReadOnlyDetails(gnbData) {
        return `
            <div class="row justify-content-center">
                <div class="col-md-8">
                    <div class="card">
                        <div class="card-header">
                            <h6 class="mb-0"><i class="fas fa-tower-broadcast me-2"></i>gNB Information</h6>
                        </div>
                        <div class="card-body">
                            <div class="row">
                                <div class="col-md-6">
                                    <div class="mb-3">
                                        <strong>gNB Name:</strong> 
                                        <div class="mt-1">
                                            <span class="badge bg-primary fs-6">${gnbData.name || 'N/A'}</span>
                                        </div>
                                    </div>
                                </div>
                                <div class="col-md-6">
                                    <div class="mb-3">
                                        <strong>TAC (Tracking Area Code):</strong>
                                        <div class="mt-1">
                                            ${gnbData.tac ? 
                                                `<span class="badge bg-secondary fs-6">${gnbData.tac}</span>` : 
                                                '<span class="text-muted">Not configured</span>'
                                            }
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
                                                <small class="text-muted">Configuration Type:</small>
                                                <div><strong>gNodeB (gNB)</strong></div>
                                            </div>
                                            <div class="col-md-6">
                                                <small class="text-muted">Network Function:</small>
                                                <div><strong>5G Base Station</strong></div>
                                            </div>
                                        </div>
                                        <div class="row mt-2">
                                            <div class="col-md-6">
                                                <small class="text-muted">TAC Range:</small>
                                                <div><strong>1 - 16,777,215</strong></div>
                                            </div>
                                            <div class="col-md-6">
                                                <small class="text-muted">Status:</small>
                                                <div>
                                                    <span class="badge bg-success">
                                                        <i class="fas fa-check-circle me-1"></i>Configured
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

    renderEditableDetails(gnbData) {
        return `
            <form id="gnbDetailsEditForm">
                <div class="row justify-content-center">
                    <div class="col-md-8">
                        <div class="card">
                            <div class="card-header">
                                <h6 class="mb-0"><i class="fas fa-edit me-2"></i>Edit gNB Information</h6>
                            </div>
                            <div class="card-body">
                                <div class="row">
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">gNB Name</label>
                                            <input type="text" class="form-control" id="edit_gnb_name" 
                                                   value="${gnbData.name || ''}" readonly>
                                            <div class="form-text">gNB name cannot be changed</div>
                                        </div>
                                    </div>
                                    <div class="col-md-6">
                                        <div class="mb-3">
                                            <label class="form-label">TAC (Tracking Area Code)</label>
                                            <input type="number" class="form-control" id="edit_gnb_tac" 
                                                   value="${gnbData.tac || ''}" placeholder="e.g., 1" min="1" max="16777215">
                                            <div class="form-text">Integer value between 1 and 16,777,215</div>
                                        </div>
                                    </div>
                                </div>
                                
                                <hr>
                                
                                <div class="bg-light p-3 rounded mb-3">
                                    <h6 class="mb-2"><i class="fas fa-info-circle me-2"></i>About TAC</h6>
                                    <p class="mb-1 small">
                                        The Tracking Area Code (TAC) is used in 5G networks to identify a tracking area, 
                                        which is a group of cells that are managed together for mobility management.
                                    </p>
                                    <p class="mb-0 small text-muted">
                                        <i class="fas fa-lightbulb me-1"></i>
                                        Leave empty if not required for your network configuration.
                                    </p>
                                </div>
                            </div>
                        </div>
                        
                        <div class="d-flex justify-content-end mt-3">
                            <button type="button" class="btn btn-secondary me-2" onclick="cancelGnbEdit()">Cancel</button>
                            <button type="button" class="btn btn-primary" onclick="saveGnbDetailsEdit()">Save Changes</button>
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
            await this.updateItem(this.currentGnbName, payload);
            
            // Refresh the details view
            await this.showDetails(this.currentGnbName);
            this.toggleEditMode(false);
            
            window.app?.notificationManager?.showNotification('gNB updated successfully!', 'success');
            
        } catch (error) {
            console.error('Failed to save gNB:', error);
            window.app?.notificationManager?.showNotification(`Failed to save gNB: ${error.message}`, 'error');
        }
    }

    getEditFormData() {
        return {
            name: document.getElementById('edit_gnb_name')?.value || '',
            tac: document.getElementById('edit_gnb_tac')?.value || ''
        };
    }

    toggleEditMode(enable = null) {
        const detailsView = document.getElementById('gnb-details-view-mode');
        const editView = document.getElementById('gnb-details-edit-mode');
        const editBtn = document.getElementById('edit-gnb-btn');
        
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
            await this.deleteItem(this.currentGnbName);
            window.app?.notificationManager?.showNotification('gNB deleted successfully!', 'success');
            
            // Navigate back to the list
            window.showSection('gnb-inventory');
            
        } catch (error) {
            console.error('Failed to delete gNB:', error);
            window.app?.notificationManager?.showNotification(`Failed to delete gNB: ${error.message}`, 'error');
        }
    }
}
