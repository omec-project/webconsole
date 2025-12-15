// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

import { BaseManager } from './baseManager.js';
import { SUBSCRIBER_API_BASE } from '../app.js';

// --- GESTOR PARA LA LISTA DE SUSCRIPTORES ---
export class SubscriberListManager extends BaseManager {
    constructor() {
        super('/subscriber', 'subscribers-list-content', SUBSCRIBER_API_BASE);
        this.type = 'subscriber';
        this.displayName = 'Subscriber';

        // List view state
        this.listState = {
            page: 1,
            limit: 20,
            plmnID: '',
            q: '',
            ueId: ''
        };
        this.listMeta = {
            page: 1,
            limit: 20,
            total: 0,
            pages: 0
        };
    }

    async loadData() {
        try {
            this.showLoading();

            const params = new URLSearchParams();
            params.set('page', String(this.listState.page));
            params.set('limit', String(this.listState.limit));
            if (this.listState.plmnID) params.set('plmnID', this.listState.plmnID);
            if (this.listState.q) params.set('q', this.listState.q);
            if (this.listState.ueId) params.set('ueId', this.listState.ueId);

            const url = `${this.apiBase}${this.apiEndpoint}?${params.toString()}`;

            const response = await fetch(url);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const body = await response.json();
            // Backend can return either legacy array or paginated object.
            const subscribersList = Array.isArray(body) ? body : (body && Array.isArray(body.items) ? body.items : []);

            if (body && !Array.isArray(body) && typeof body === 'object') {
                this.listMeta = {
                    page: Number(body.page) || this.listState.page,
                    limit: Number(body.limit) || this.listState.limit,
                    total: Number(body.total) || 0,
                    pages: Number(body.pages) || 0
                };
            } else {
                this.listMeta = {
                    page: 1,
                    limit: subscribersList.length,
                    total: subscribersList.length,
                    pages: 1
                };
            }

            this.data = subscribersList;
            this.render(subscribersList);
            
        } catch (error) {
            this.showError(`Failed to load subscribers: ${error.message}`);
            console.error('Load subscribers error:', error);
        }
    }

    setListState(partial) {
        this.listState = {
            ...this.listState,
            ...partial
        };
    }

    goToPage(page) {
        const newPage = Math.max(1, parseInt(page, 10) || 1);
        this.setListState({ page: newPage });
        return this.loadData();
    }

    applyListControls() {
        const qEl = document.getElementById('subscriber-list-q');
        const ueIdEl = document.getElementById('subscriber-list-ueid');
        const plmnEl = document.getElementById('subscriber-list-plmn');
        const limitEl = document.getElementById('subscriber-list-limit');

        const q = qEl ? qEl.value.trim() : '';
        const ueId = ueIdEl ? ueIdEl.value.trim() : '';
        const plmnID = plmnEl ? plmnEl.value.trim() : '';
        const limit = limitEl ? parseInt(limitEl.value, 10) : this.listState.limit;

        this.setListState({
            q,
            ueId,
            plmnID,
            limit: Number.isFinite(limit) && limit > 0 ? limit : this.listState.limit,
            page: 1
        });
        return this.loadData();
    }

    clearListControls() {
        this.setListState({ page: 1, limit: this.listState.limit, plmnID: '', q: '', ueId: '' });
        return this.loadData();
    }

    renderListControls() {
        const qValue = this.listState.q || '';
        const ueIdValue = this.listState.ueId || '';
        const plmnValue = this.listState.plmnID || '';
        const limitValue = String(this.listState.limit);
        const page = this.listMeta.page || this.listState.page;
        const pages = this.listMeta.pages || 0;
        const total = this.listMeta.total || 0;

        const prevDisabled = page <= 1 ? 'disabled' : '';
        const nextDisabled = pages > 0 && page >= pages ? 'disabled' : '';

        return `
            <div class="card mb-3">
                <div class="card-body">
                    <div class="row g-2 align-items-end">
                        <div class="col-md-4">
                            <label class="form-label">Search (contains)</label>
                            <input type="text" class="form-control" id="subscriber-list-q" placeholder="e.g., 20893" value="${qValue}">
                        </div>
                        <div class="col-md-4">
                            <label class="form-label">IMSI / UE ID (exact)</label>
                            <input type="text" class="form-control" id="subscriber-list-ueid" placeholder="e.g., imsi-208930100007487" value="${ueIdValue}">
                        </div>
                        <div class="col-md-2">
                            <label class="form-label">PLMN ID</label>
                            <input type="text" class="form-control" id="subscriber-list-plmn" placeholder="5 or 6 digits" value="${plmnValue}">
                        </div>
                        <div class="col-md-2">
                            <label class="form-label">Page size</label>
                            <select class="form-select" id="subscriber-list-limit">
                                <option value="10" ${limitValue === '10' ? 'selected' : ''}>10</option>
                                <option value="20" ${limitValue === '20' ? 'selected' : ''}>20</option>
                                <option value="50" ${limitValue === '50' ? 'selected' : ''}>50</option>
                                <option value="100" ${limitValue === '100' ? 'selected' : ''}>100</option>
                            </select>
                        </div>
                    </div>
                    <div class="d-flex gap-2 mt-3">
                        <button class="btn btn-primary" id="subscriber-list-apply">Apply</button>
                        <button class="btn btn-outline-secondary" id="subscriber-list-clear">Clear</button>
                    </div>
                    <hr />
                    <div class="d-flex justify-content-between align-items-center">
                        <div class="text-muted">Total: ${total}${pages ? ` | Page ${page} of ${pages}` : ''}</div>
                        <div class="btn-group" role="group" aria-label="Pagination">
                            <button class="btn btn-outline-secondary" id="subscriber-list-prev" ${prevDisabled}>
                                <i class="fas fa-chevron-left"></i> Prev
                            </button>
                            <button class="btn btn-outline-secondary" id="subscriber-list-next" ${nextDisabled}>
                                Next <i class="fas fa-chevron-right"></i>
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    bindListControls() {
        const applyBtn = document.getElementById('subscriber-list-apply');
        const clearBtn = document.getElementById('subscriber-list-clear');
        const prevBtn = document.getElementById('subscriber-list-prev');
        const nextBtn = document.getElementById('subscriber-list-next');
        const qEl = document.getElementById('subscriber-list-q');
        const ueIdEl = document.getElementById('subscriber-list-ueid');

        if (applyBtn) applyBtn.addEventListener('click', () => this.applyListControls());
        if (clearBtn) clearBtn.addEventListener('click', () => this.clearListControls());
        if (prevBtn) prevBtn.addEventListener('click', () => this.goToPage((this.listMeta.page || this.listState.page) - 1));
        if (nextBtn) nextBtn.addEventListener('click', () => this.goToPage((this.listMeta.page || this.listState.page) + 1));

        const onEnter = (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                this.applyListControls();
            }
        };
        if (qEl) qEl.addEventListener('keydown', onEnter);
        if (ueIdEl) ueIdEl.addEventListener('keydown', onEnter);
    }

    render(subscribers) {
        const container = document.getElementById(this.containerId);
		if (!container) {
			return;
		}

		let html = this.renderListControls();
        
        if (!subscribers || subscribers.length === 0) {
			html += `
				<div class="alert alert-info">
					<i class="fas fa-info-circle me-2"></i>
					No subscribers found
				</div>
			`;
			container.innerHTML = html;
			this.bindListControls();
            return;
        }

        html += '<div class="table-responsive"><table class="table table-striped">';
        html += '<thead><tr><th>UE ID (IMSI)</th><th>PLMN ID</th><th>Actions</th></tr></thead><tbody>';
        
        subscribers.forEach(subscriber => {
            const ueId = subscriber.ueId || 'N/A';
            const plmnId = subscriber.plmnID || 'N/A';
            
            html += `
                <tr class="subscriber-row" onclick="showSubscriberDetails('${ueId}')" style="cursor: pointer;">
                    <td><strong>${ueId}</strong></td>
                    <td><code>${plmnId}</code></td>
                    <td onclick="event.stopPropagation();">
                        <button class="btn btn-sm btn-outline-primary me-1" 
                                onclick="editItem('${this.type}', '${ueId}')">
                            <i class="fas fa-edit"></i> Edit
                        </button>
                        <button class="btn btn-sm btn-outline-danger" 
                                onclick="deleteItem('${this.type}', '${ueId}')">
                            <i class="fas fa-trash"></i> Delete
                        </button>
                    </td>
                </tr>
            `;
        });
        

        html += '</tbody></table></div>';
        container.innerHTML = html;
		this.bindListControls();
    }

    getFormFields(isEdit = false) {
        return `
            <div class="mb-3">
                <label class="form-label">UE ID (IMSI)</label>
                <input type="text" class="form-control" id="sub_ueId" 
                       ${isEdit ? 'readonly' : ''} placeholder="e.g., imsi-208930100007487" required>
                <div class="form-text">International Mobile Subscriber Identity</div>
            </div>
            <div class="mb-3">
                <label class="form-label">PLMN ID</label>
                <input type="text" class="form-control" id="sub_plmnID" 
                       placeholder="5 or 6 digits" pattern="\\d{5,6}" maxlength="6" required>
                <div class="form-text">Public Land Mobile Network ID</div>
            </div>
            <div class="mb-3">
                <label class="form-label">Key (Ki)</label>
                <input type="text" class="form-control" id="sub_key" 
                       placeholder="Hexadecimal characters" pattern="[0-9a-fA-F]+" required>
                <div class="form-text">Authentication key (hexadecimal characters)</div>
            </div>
            <div class="mb-3">
                <label class="form-label">OPc</label>
                <input type="text" class="form-control" id="sub_opc" 
                       placeholder="Hexadecimal characters" pattern="[0-9a-fA-F]+" required>
                <div class="form-text">Operator key (hexadecimal characters)</div>
            </div>
            <div class="mb-3">
                <label class="form-label">Sequence Number (SQN)</label>
                <input type="text" class="form-control" id="sub_sequenceNumber" 
                       placeholder="e.g., 16f3b3f70fc2" required>
                <div class="form-text">Authentication sequence number</div>
            </div>
            <div class="row">
                <div class="col-md-6 mb-3">
                    <label class="form-label">K4 SNO</label>
                    <select class="form-select" id="sub_k4_sno">
                        <option value="">Loading K4 keys...</option>
                    </select>
                    <div class="form-text">K4 Serial Number reference (optional)</div>
                </div>
                <div class="col-md-6 mb-3">
                    <label class="form-label">Encryption Algorithm</label>
                    <input type="number" class="form-control" id="sub_encryptionAlgorithm" 
                           placeholder="e.g., 0" value="0" min="0">
                    <div class="form-text">Algorithm identifier for encryption (optional)</div>
                </div>
            </div>
        `;
    }

    validateFormData(data) {
        const errors = [];
        
        if (!data.sub_ueId || data.sub_ueId.trim() === '') {
            errors.push('UE ID is required');
        }
        
        if (!data.sub_plmnID || !/^\d{5,6}$/.test(data.sub_plmnID)) {
            errors.push('PLMN ID must be 5 or 6 digits');
        }
        
        if (!data.sub_key || !/^[0-9a-fA-F]+$/.test(data.sub_key)) {
            errors.push('Key (Ki) must contain only hexadecimal characters');
        }
        
        if (!data.sub_opc || !/^[0-9a-fA-F]+$/.test(data.sub_opc)) {
            errors.push('OPc must contain only hexadecimal characters');
        }
        
        if (!data.sub_sequenceNumber || data.sub_sequenceNumber.trim() === '') {
            errors.push('Sequence Number is required');
        }
        
        return {
            isValid: errors.length === 0,
            errors: errors
        };
    }

    preparePayload(formData, isEdit = false) {
        // Map form data to API structure - SubsOverrideData
        const payload = {
            ueId: formData.sub_ueId,
            plmnID: formData.sub_plmnID,
            OPc: formData.sub_opc,
            Key: formData.sub_key,
            SequenceNumber: formData.sub_sequenceNumber,
            EncryptionAlgorithm: parseInt(formData.sub_encryptionAlgorithm) || 0
        };

        // Add K4 SNO if provided
        if (formData.sub_k4_sno && formData.sub_k4_sno !== '') {
            payload.k4_sno = parseInt(formData.sub_k4_sno);
        }

        return payload;
    }

    async createItem(itemData, ueId = null) {
        try {
            // If ueId is not provided as second parameter, extract it from itemData
            const actualUeId = ueId || itemData.ueId;
            
            if (!actualUeId) {
                throw new Error('UE ID is required for subscriber creation');
            }
            
            console.log('Creating subscriber with UE ID:', actualUeId);
            console.log('Payload:', itemData);
            
            const response = await fetch(`${this.apiBase}${this.apiEndpoint}/${encodeURIComponent(actualUeId)}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(itemData)
            });

            if (!response.ok) {
                const errorText = await response.text();
                console.error('API Error Response:', errorText);
                throw new Error(errorText || `HTTP ${response.status}`);
            }

            return response.status === 201 ? {} : await response.json();
        } catch (error) {
            console.error('Create item error:', error);
            throw error;
        }
    }

    async updateItem(ueId, itemData) {
        try {
            const response = await fetch(`${this.apiBase}${this.apiEndpoint}/${encodeURIComponent(ueId)}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(itemData)
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(errorText || `HTTP ${response.status}`);
            }

            return response.status === 204 ? {} : await response.json();
        } catch (error) {
            throw error;
        }
    }

    async deleteItem(ueId) {
        try {
            const response = await fetch(`${this.apiBase}${this.apiEndpoint}/${encodeURIComponent(ueId)}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(errorText || `HTTP ${response.status}`);
            }

            return response.status === 204 ? {} : await response.json();
        } catch (error) {
            throw error;
        }
    }

    async loadK4Keys() {
        try {
            const response = await fetch(`${SUBSCRIBER_API_BASE}/k4opt`);
            if (response.ok) {
                const k4Keys = await response.json();
                const select = document.getElementById('sub_k4_sno');
                if (select && Array.isArray(k4Keys)) {
                    select.innerHTML = '<option value="">None (Optional)</option>';
                    k4Keys.forEach(key => {
                        const option = document.createElement('option');
                        option.value = key.k4_sno;
                        option.textContent = `SNO ${key.k4_sno} - Key: ${key.k4?.substring(0, 8)}...`;
                        select.appendChild(option);
                    });
                } else {
                    select.innerHTML = '<option value="">No K4 keys available</option>';
                }
            } else {
                const select = document.getElementById('sub_k4_sno');
                if (select) {
                    select.innerHTML = '<option value="">Error loading K4 keys</option>';
                }
            }
        } catch (error) {
            console.warn('Failed to load K4 keys:', error.message);
            const select = document.getElementById('sub_k4_sno');
            if (select) {
                select.innerHTML = '<option value="">Error loading K4 keys</option>';
            }
        }
    }

    async loadK4KeysForEdit() {
        try {
            const response = await fetch(`${SUBSCRIBER_API_BASE}/k4opt`);
            if (response.ok) {
                const k4Keys = await response.json();
                const select = document.getElementById('edit_sub_k4_sno');
                if (select && Array.isArray(k4Keys)) {
                    select.innerHTML = '<option value="">None (Optional)</option>';
                    k4Keys.forEach(key => {
                        const option = document.createElement('option');
                        option.value = key.k4_sno;
                        option.textContent = `SNO ${key.k4_sno} - Key: ${key.k4?.substring(0, 8)}...`;
                        select.appendChild(option);
                    });
                } else {
                    select.innerHTML = '<option value="">No K4 keys available</option>';
                }
            } else {
                const select = document.getElementById('edit_sub_k4_sno');
                if (select) {
                    select.innerHTML = '<option value="">Error loading K4 keys</option>';
                }
            }
        } catch (error) {
            console.warn('Failed to load K4 keys for edit:', error.message);
            const select = document.getElementById('edit_sub_k4_sno');
            if (select) {
                select.innerHTML = '<option value="">Error loading K4 keys</option>';
            }
        }
    }

    async loadItemData(ueId) {
        try {
            const response = await fetch(`${this.apiBase}${this.apiEndpoint}/${encodeURIComponent(ueId)}`);
            if (response.ok) {
                const subsData = await response.json();
                
                // Populate basic fields
                this.setFieldValue('sub_ueId', subsData.ueId);
                this.setFieldValue('sub_plmnID', subsData.plmnID);
                
                // Extract authentication data if available
                if (subsData.AuthenticationSubscription) {
                    const authData = subsData.AuthenticationSubscription;
                    this.setFieldValue('sub_key', authData.PermanentKey?.PermanentKeyValue);
                    this.setFieldValue('sub_opc', authData.Opc?.OpcValue);
                    this.setFieldValue('sub_sequenceNumber', authData.SequenceNumber);
                    
                    // Set encryption algorithm if available
                    if (authData.Opc?.EncryptionAlgorithm !== undefined) {
                        this.setFieldValue('sub_encryptionAlgorithm', authData.Opc.EncryptionAlgorithm);
                    }
                }
            }
        } catch (error) {
            console.error('Failed to load subscriber data:', error);
        }
    }

    setFieldValue(fieldId, value) {
        const field = document.getElementById(fieldId);
        if (field && value !== undefined && value !== null) {
            field.value = value;
        }
    }

    // New methods for details view
    async showDetails(ueId) {
        try {
            const response = await fetch(`${this.apiBase}${this.apiEndpoint}/${encodeURIComponent(ueId)}`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const subscriberData = await response.json();
            this.currentSubscriberData = subscriberData;
            this.currentSubscriberUeId = ueId;
            this.renderDetailsView(subscriberData);
            
        } catch (error) {
            console.error('Failed to load subscriber details:', error);
            // Show error notification
            window.app?.notificationManager?.showNotification('Error loading subscriber details', 'error');
        }
    }

    renderDetailsView(subscriberData) {
        const container = document.getElementById('subscriber-details-content');
        const title = document.getElementById('subscriber-detail-title');
        
        if (!container || !title) {
            console.error('Details container not found');
            return;
        }

        const ueId = subscriberData.ueId || 'Unknown';
        title.textContent = `Subscriber: ${ueId}`;

        const html = `
            <div id="subscriber-details-view-mode">
                ${this.renderReadOnlyDetails(subscriberData)}
            </div>
            <div id="subscriber-details-edit-mode" style="display: none;">
                ${this.renderEditableDetails(subscriberData)}
            </div>
        `;

        container.innerHTML = html;
    }

    renderReadOnlyDetails(subscriberData) {
        const authData = subscriberData.AuthenticationSubscription || {};
        const amData = subscriberData.AccessAndMobilitySubscriptionData || {};
        const smData = subscriberData.SessionManagementSubscriptionData || [];
        
        // Extraer todos los componentes de autenticación
        const permanentKey = authData.permanentKey || {};
        const milenage = authData.milenage || {};
        const op = milenage.op || {};
        const rotations = milenage.rotations || {};
        const constants = milenage.constants || {};
        const tuak = authData.tuak || {};
        const top = tuak.top || {};
        const opc = authData.opc || {};
        const topc = authData.topc || {};

        // Extraer componentes de Access and Mobility
        const ambr = amData.subscribedUeAmbr || {};
        const nssai = amData.nssai || {};
        const serviceAreaRestriction = amData.serviceAreaRestriction || {};
        const sorInfo = amData.sorInfo || {};
        
        return `
            <div class="row">
                <!-- Información Básica -->
                <div class="col-md-12">
                    <div class="card mb-3">
                        <div class="card-header bg-primary text-white">
                            <h5 class="mb-0"><i class="fas fa-user-shield me-2"></i>Authentication Information</h5>
                        </div>
                        <div class="card-body">
                            <!-- Basic Auth Info -->
                            <div class="row mb-4">
                                <div class="col-md-6">
                                    <h6 class="border-bottom pb-2"><i class="fas fa-info-circle me-2"></i>Basic Details</h6>
                                    <div class="mb-2">
                                        <strong>UE ID:</strong> 
                                        <code class="fs-6">${subscriberData.ueId || 'N/A'}</code>
                                    </div>
                                    <div class="mb-2">
                                        <strong>Authentication Method:</strong> 
                                        <span class="badge bg-info">${authData.authenticationMethod || 'N/A'}</span>
                                    </div>
                                    <div class="mb-2">
                                        <strong>Sequence Number:</strong> 
                                        <code>${authData.sequenceNumber || 'N/A'}</code>
                                    </div>
                                    <div class="mb-2">
                                        <strong>Auth Management Field:</strong> 
                                        <span class="badge bg-secondary">${authData.authenticationManagementField || 'N/A'}</span>
                                    </div>
                                    <div class="mb-2">
                                        <strong>Vector Algorithm:</strong> 
                                        <span class="badge bg-primary">${authData.vectorAlgorithm || 'N/A'}</span>
                                    </div>
                                    <div class="mb-2">
                                        <strong>K4 SNO:</strong> 
                                        <code>${authData.k4_sno !== undefined ? authData.k4_sno : 'N/A'}</code>
                                    </div>
                                </div>
                                
                                <!-- Permanent Key Info -->
                                <div class="col-md-6">
                                    <h6 class="border-bottom pb-2"><i class="fas fa-key me-2"></i>Permanent Key</h6>
                                    <div class="mb-2">
                                        <strong>Key Value:</strong>
                                        <div class="mt-1">
                                            <code class="text-break">${permanentKey.permanentKeyValue || 'N/A'}</code>
                                        </div>
                                    </div>
                                    <div class="mb-2">
                                        <strong>Tag:</strong>
                                        <code>${permanentKey.tag && permanentKey.tag.trim() !== '' ? permanentKey.tag : 'N/A'}</code>
                                    </div>
                                    <div class="mb-2">
                                        <strong>Encryption Key:</strong>
                                        <code>${permanentKey.encryptionKey || 'N/A'}</code>
                                    </div>
                                    <div class="mb-2">
                                        <strong>Encryption Algorithm:</strong>
                                        <span class="badge bg-info">${permanentKey.encryptionAlgorithm || 'N/A'}</span>
                                    </div>
                                </div>
                            </div>

                            <!-- Milenage Section -->
                            <div class="row mb-4">
                                <div class="col-12">
                                    <h6 class="border-bottom pb-2"><i class="fas fa-cogs me-2"></i>Milenage Configuration</h6>
                                </div>
                                <!-- OP Values -->
                                <div class="col-md-4">
                                    <div class="card bg-light">
                                        <div class="card-body">
                                            <h6 class="card-title">OP Values</h6>
                                            <div class="mb-2">
                                                <strong>OP Value:</strong>
                                                <div><code class="text-break">${op.opValue || 'N/A'}</code></div>
                                            </div>
                                            <div class="mb-2">
                                                <strong>Encryption Key:</strong>
                                                <div><code>${op.encryptionKey || 'N/A'}</code></div>
                                            </div>
                                            <div class="mb-2">
                                                <strong>Encryption Algorithm:</strong>
                                                <div><span class="badge bg-info">${op.encryptionAlgorithm || 'N/A'}</span></div>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                <!-- Rotations -->
                                <div class="col-md-4">
                                    <div class="card bg-light">
                                        <div class="card-body">
                                            <h6 class="card-title">Rotations</h6>
                                            <div class="mb-1"><strong>R1:</strong> <code>${rotations.r1 || 'N/A'}</code></div>
                                            <div class="mb-1"><strong>R2:</strong> <code>${rotations.r2 || 'N/A'}</code></div>
                                            <div class="mb-1"><strong>R3:</strong> <code>${rotations.r3 || 'N/A'}</code></div>
                                            <div class="mb-1"><strong>R4:</strong> <code>${rotations.r4 || 'N/A'}</code></div>
                                            <div class="mb-1"><strong>R5:</strong> <code>${rotations.r5 || 'N/A'}</code></div>
                                        </div>
                                    </div>
                                </div>
                                <!-- Constants -->
                                <div class="col-md-4">
                                    <div class="card bg-light">
                                        <div class="card-body">
                                            <h6 class="card-title">Constants</h6>
                                            <div class="mb-1"><strong>C1:</strong> <code>${constants.c1 || 'N/A'}</code></div>
                                            <div class="mb-1"><strong>C2:</strong> <code>${constants.c2 || 'N/A'}</code></div>
                                            <div class="mb-1"><strong>C3:</strong> <code>${constants.c3 || 'N/A'}</code></div>
                                            <div class="mb-1"><strong>C4:</strong> <code>${constants.c4 || 'N/A'}</code></div>
                                            <div class="mb-1"><strong>C5:</strong> <code>${constants.c5 || 'N/A'}</code></div>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            <!-- Tuak and OPC Section -->
                            <div class="row">
                                <div class="col-md-6">
                                    <h6 class="border-bottom pb-2"><i class="fas fa-shield-alt me-2"></i>TUAK Configuration</h6>
                                    <div class="card bg-light mb-3">
                                        <div class="card-body">
                                            <div class="mb-2">
                                                <strong>TOP Value:</strong>
                                                <div><code class="text-break">${top.topValue || 'N/A'}</code></div>
                                            </div>
                                            <div class="mb-2">
                                                <strong>TOP Encryption Key:</strong>
                                                <div><code>${top.encryptionKey || 'N/A'}</code></div>
                                            </div>
                                            <div class="mb-2">
                                                <strong>TOP Encryption Algorithm:</strong>
                                                <div><span class="badge bg-info">${top.encryptionAlgorithm || 'N/A'}</span></div>
                                            </div>
                                            <div class="mb-2">
                                                <strong>Keccak Iterations:</strong>
                                                <div><code>${tuak.keccakIterations || 'N/A'}</code></div>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                
                                <div class="col-md-6">
                                    <h6 class="border-bottom pb-2"><i class="fas fa-lock me-2"></i>OPC/TOPC Information</h6>
                                    <!-- OPC Info -->
                                    <div class="card bg-light mb-3">
                                        <div class="card-body">
                                            <h6 class="card-title">OPC Details</h6>
                                            <div class="mb-2">
                                                <strong>OPC Value:</strong>
                                                <div><code class="text-break">${opc.opcValue || 'N/A'}</code></div>
                                            </div>
                                            <div class="mb-2">
                                                <strong>Encryption Key:</strong>
                                                <div><code>${opc.encryptionKey || 'N/A'}</code></div>
                                            </div>
                                            <div class="mb-2">
                                                <strong>Encryption Algorithm:</strong>
                                                <div><span class="badge bg-info">${opc.encryptionAlgorithm || 'N/A'}</span></div>
                                            </div>
                                        </div>
                                    </div>
                                    <!-- TOPC Info -->
                                    <div class="card bg-light">
                                        <div class="card-body">
                                            <h6 class="card-title">TOPC Details</h6>
                                            <div class="mb-2">
                                                <strong>TOPC Value:</strong>
                                                <div><code class="text-break">${topc.topcValue || 'N/A'}</code></div>
                                            </div>
                                            <div class="mb-2">
                                                <strong>Encryption Key:</strong>
                                                <div><code>${topc.encryptionKey || 'N/A'}</code></div>
                                            </div>
                                            <div class="mb-2">
                                                <strong>Encryption Algorithm:</strong>
                                                <div><span class="badge bg-info">${topc.encryptionAlgorithm || 'N/A'}</span></div>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Session Management Section -->
            <div class="row mt-4">
                <div class="col-12">
                    <div class="card mb-3">
                        <div class="card-header bg-info text-white">
                            <h5 class="mb-0"><i class="fas fa-server me-2"></i>Session Management</h5>
                        </div>
                        <div class="card-body">
                            ${smData.map((session, index) => `
                                <div class="session-block ${index > 0 ? 'mt-4 pt-4 border-top' : ''}">
                                    <!-- Single NSSAI Information -->
                                    <div class="row mb-3">
                                        <div class="col-12">
                                            <h6 class="border-bottom pb-2">
                                                <i class="fas fa-layer-group me-2"></i>Network Slice ${index + 1}
                                            </h6>
                                            <div class="mb-3">
                                                <strong>SST:</strong>
                                                <code>${session.singleNssai?.sst || 'N/A'}</code>
                                                ${session.singleNssai?.sd ? 
                                                    `<br><strong>SD:</strong> <code>${session.singleNssai.sd}</code>` : 
                                                    ''}
                                            </div>
                                        </div>
                                    </div>

                                    <!-- DNN Configurations -->
                                    ${Object.entries(session.dnnConfigurations || {}).map(([dnn, config]) => `
                                        <div class="card bg-light mb-3">
                                            <div class="card-header">
                                                <h6 class="mb-0">
                                                    <i class="fas fa-network-wired me-2"></i>DNN: ${dnn}
                                                </h6>
                                            </div>
                                            <div class="card-body">
                                                <!-- PDU Session Types -->
                                                <div class="row mb-3">
                                                    <div class="col-md-6">
                                                        <h6 class="card-subtitle mb-2">PDU Session Types</h6>
                                                        <div class="mb-2">
                                                            <strong>Default:</strong>
                                                            <span class="badge bg-primary">${config.pduSessionTypes?.defaultSessionType || 'N/A'}</span>
                                                        </div>
                                                        <div>
                                                            <strong>Allowed:</strong><br>
                                                            ${config.pduSessionTypes?.allowedSessionTypes?.map(type =>
                                                                `<span class="badge bg-info me-1">${type}</span>`
                                                            ).join('') || 'N/A'}
                                                        </div>
                                                    </div>
                                                    <div class="col-md-6">
                                                        <h6 class="card-subtitle mb-2">SSC Modes</h6>
                                                        <div class="mb-2">
                                                            <strong>Default:</strong>
                                                            <span class="badge bg-primary">${config.sscModes?.defaultSscMode || 'N/A'}</span>
                                                        </div>
                                                        <div>
                                                            <strong>Allowed:</strong><br>
                                                            ${config.sscModes?.allowedSscModes?.map(mode =>
                                                                `<span class="badge bg-info me-1">${mode}</span>`
                                                            ).join('') || 'N/A'}
                                                        </div>
                                                    </div>
                                                </div>

                                                <!-- QoS Profile -->
                                                <div class="row mb-3">
                                                    <div class="col-12">
                                                        <h6 class="card-subtitle mb-2">5G QoS Profile</h6>
                                                        <div class="card bg-white">
                                                            <div class="card-body">
                                                                <div class="row">
                                                                    <div class="col-md-4">
                                                                        <strong>5QI:</strong>
                                                                        <code>${config.Var5gQosProfile?.Var5qi || 'N/A'}</code>
                                                                    </div>
                                                                    <div class="col-md-4">
                                                                        <strong>Priority Level:</strong>
                                                                        <code>${config.Var5gQosProfile?.priorityLevel || 'N/A'}</code>
                                                                    </div>
                                                                    <div class="col-md-4">
                                                                        <strong>ARP Priority:</strong>
                                                                        <code>${config.Var5gQosProfile?.arp?.priorityLevel || 'N/A'}</code>
                                                                    </div>
                                                                </div>
                                                                <div class="row mt-2">
                                                                    <div class="col-md-6">
                                                                        <strong>Preemption Capability:</strong>
                                                                        <span class="badge ${config.Var5gQosProfile?.arp?.preemptCap === 'MAY_PREEMPT' ? 'bg-warning' : 'bg-secondary'}">
                                                                            ${config.Var5gQosProfile?.arp?.preemptCap || 'N/A'}
                                                                        </span>
                                                                    </div>
                                                                    <div class="col-md-6">
                                                                        <strong>Preemption Vulnerability:</strong>
                                                                        <span class="badge ${config.Var5gQosProfile?.arp?.preemptVuln === 'PREEMPTABLE' ? 'bg-warning' : 'bg-secondary'}">
                                                                            ${config.Var5gQosProfile?.arp?.preemptVuln || 'N/A'}
                                                                        </span>
                                                                    </div>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    </div>
                                                </div>

                                                <!-- Session AMBR -->
                                                <div class="row mb-3">
                                                    <div class="col-12">
                                                        <h6 class="card-subtitle mb-2">Session AMBR</h6>
                                                        <div class="row">
                                                            <div class="col-md-6">
                                                                <strong>Uplink:</strong>
                                                                <code>${config.sessionAmbr?.uplink || 'N/A'}</code>
                                                            </div>
                                                            <div class="col-md-6">
                                                                <strong>Downlink:</strong>
                                                                <code>${config.sessionAmbr?.downlink || 'N/A'}</code>
                                                            </div>
                                                        </div>
                                                    </div>
                                                </div>

                                                <!-- IP Addresses and Security -->
                                                <div class="row">
                                                    <div class="col-md-6">
                                                        <h6 class="card-subtitle mb-2">Static IP Addresses</h6>
                                                        ${config.staticIpAddress?.map(ip => `
                                                            <div class="mb-1">
                                                                ${ip.ipv4Addr ? `<div>IPv4: <code>${ip.ipv4Addr}</code></div>` : ''}
                                                                ${ip.ipv6Addr ? `<div>IPv6: <code>${ip.ipv6Addr}</code></div>` : ''}
                                                                ${ip.ipv6Prefix ? `<div>IPv6 Prefix: <code>${ip.ipv6Prefix}</code></div>` : ''}
                                                            </div>
                                                        `).join('') || '<div class="text-muted">No static IPs configured</div>'}
                                                    </div>
                                                    <div class="col-md-6">
                                                        <h6 class="card-subtitle mb-2">UP Security</h6>
                                                        <div class="mb-2">
                                                            <strong>Integrity:</strong>
                                                            <span class="badge ${config.upSecurity?.upIntegr === 'REQUIRED' ? 'bg-success' : 
                                                                              config.upSecurity?.upIntegr === 'PREFERRED' ? 'bg-warning' : 'bg-secondary'}">
                                                                ${config.upSecurity?.upIntegr || 'N/A'}
                                                            </span>
                                                        </div>
                                                        <div>
                                                            <strong>Confidentiality:</strong>
                                                            <span class="badge ${config.upSecurity?.upConfid === 'REQUIRED' ? 'bg-success' : 
                                                                              config.upSecurity?.upConfid === 'PREFERRED' ? 'bg-warning' : 'bg-secondary'}">
                                                                ${config.upSecurity?.upConfid || 'N/A'}
                                                            </span>
                                                        </div>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    `).join('')}

                                    <!-- Internal Groups and Shared Configurations -->
                                    <div class="row">
                                        <div class="col-md-6">
                                            <h6 class="border-bottom pb-2">Internal Group IDs</h6>
                                            ${session.internalGroupIds && session.internalGroupIds.length > 0 ?
                                                session.internalGroupIds.map(id =>
                                                    `<span class="badge bg-secondary me-1 mb-1">${id}</span>`
                                                ).join('') :
                                                '<div class="text-muted">No internal groups</div>'
                                            }
                                        </div>
                                        <div class="col-md-6">
                                            <h6 class="border-bottom pb-2">Shared DNN Configurations</h6>
                                            ${session.sharedDnnConfigurationsIds ?
                                                `<span class="badge bg-info">${session.sharedDnnConfigurationsIds}</span>` :
                                                '<div class="text-muted">No shared configurations</div>'
                                            }
                                        </div>
                                    </div>
                                </div>
                            `).join('')}
                        </div>
                    </div>
                </div>
            </div>

            <!-- Access and Mobility Section -->
            <div class="row mt-4">
                <div class="col-12">
                    <div class="card mb-3">
                        <div class="card-header bg-success text-white">
                            <h5 class="mb-0"><i class="fas fa-network-wired me-2"></i>Access and Mobility Information</h5>
                        </div>
                        <div class="card-body">
                            <!-- Basic Features -->
                            <div class="row mb-4">
                                <div class="col-md-6">
                                    <h6 class="border-bottom pb-2"><i class="fas fa-cog me-2"></i>Basic Features</h6>
                                    <div class="mb-2">
                                        <strong>Supported Features:</strong>
                                        <code>${amData.supportedFeatures || 'N/A'}</code>
                                    </div>
                                    <div class="mb-2">
                                        <strong>GPSIs:</strong>
                                        <div class="mt-1">
                                            ${amData.gpsis && amData.gpsis.length > 0 ? 
                                                amData.gpsis.map(gpsi => 
                                                    `<span class="badge bg-info me-1">${gpsi}</span>`
                                                ).join('') : 
                                                '<span class="text-muted">No GPSIs defined</span>'
                                            }
                                        </div>
                                    </div>
                                    <div class="mb-2">
                                        <strong>Internal Group IDs:</strong>
                                        <div class="mt-1">
                                            ${amData.internalGroupIds && amData.internalGroupIds.length > 0 ? 
                                                amData.internalGroupIds.map(id => 
                                                    `<span class="badge bg-secondary me-1">${id}</span>`
                                                ).join('') : 
                                                '<span class="text-muted">No internal groups</span>'
                                            }
                                        </div>
                                    </div>
                                </div>

                                <div class="col-md-6">
                                    <h6 class="border-bottom pb-2"><i class="fas fa-tachometer-alt me-2"></i>AMBR Settings</h6>
                                    <div class="card bg-light">
                                        <div class="card-body">
                                            <div class="mb-2">
                                                <strong>Uplink:</strong>
                                                <code>${ambr.uplink || 'N/A'}</code>
                                            </div>
                                            <div class="mb-2">
                                                <strong>Downlink:</strong>
                                                <code>${ambr.downlink || 'N/A'}</code>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            <!-- Network Slicing Section -->
                            <div class="row mb-4">
                                <div class="col-12">
                                    <h6 class="border-bottom pb-2"><i class="fas fa-layer-group me-2"></i>Network Slicing (NSSAI)</h6>
                                </div>
                                <div class="col-md-6">
                                    <div class="card bg-light">
                                        <div class="card-body">
                                            <h6 class="card-title">Default Single NSSAIs</h6>
                                            ${nssai.defaultSingleNssais && nssai.defaultSingleNssais.length > 0 ? 
                                                nssai.defaultSingleNssais.map(snssai => `
                                                    <div class="mb-2 p-2 border rounded">
                                                        <div><strong>SST:</strong> <code>${snssai.sst}</code></div>
                                                        ${snssai.sd ? `<div><strong>SD:</strong> <code>${snssai.sd}</code></div>` : ''}
                                                    </div>
                                                `).join('') : 
                                                '<div class="text-muted">No default NSSAIs defined</div>'
                                            }
                                        </div>
                                    </div>
                                </div>
                                <div class="col-md-6">
                                    <div class="card bg-light">
                                        <div class="card-body">
                                            <h6 class="card-title">Single NSSAIs</h6>
                                            ${nssai.singleNssais && nssai.singleNssais.length > 0 ? 
                                                nssai.singleNssais.map(snssai => `
                                                    <div class="mb-2 p-2 border rounded">
                                                        <div><strong>SST:</strong> <code>${snssai.sst}</code></div>
                                                        ${snssai.sd ? `<div><strong>SD:</strong> <code>${snssai.sd}</code></div>` : ''}
                                                    </div>
                                                `).join('') : 
                                                '<div class="text-muted">No single NSSAIs defined</div>'
                                            }
                                        </div>
                                    </div>
                                </div>
                            </div>

                            <!-- Restrictions and Areas -->
                            <div class="row mb-4">
                                <div class="col-md-6">
                                    <h6 class="border-bottom pb-2"><i class="fas fa-ban me-2"></i>Network Restrictions</h6>
                                    <div class="mb-3">
                                        <strong>RAT Restrictions:</strong>
                                        <div class="mt-1">
                                            ${amData.ratRestrictions && amData.ratRestrictions.length > 0 ? 
                                                amData.ratRestrictions.map(rat => 
                                                    `<span class="badge bg-warning text-dark me-1">${rat}</span>`
                                                ).join('') : 
                                                '<span class="text-muted">No RAT restrictions</span>'
                                            }
                                        </div>
                                    </div>
                                    <div class="mb-3">
                                        <strong>Core Network Types:</strong>
                                        <div class="mt-1">
                                            ${amData.coreNetworkTypeRestrictions && amData.coreNetworkTypeRestrictions.length > 0 ? 
                                                amData.coreNetworkTypeRestrictions.map(type => 
                                                    `<span class="badge bg-primary me-1">${type}</span>`
                                                ).join('') : 
                                                '<span class="text-muted">No core network restrictions</span>'
                                            }
                                        </div>
                                    </div>
                                </div>

                                <div class="col-md-6">
                                    <h6 class="border-bottom pb-2"><i class="fas fa-map-marker-alt me-2"></i>Service Area Restrictions</h6>
                                    <div class="card bg-light">
                                        <div class="card-body">
                                            <div class="mb-2">
                                                <strong>Restriction Type:</strong>
                                                <span class="badge ${serviceAreaRestriction.restrictionType === 'ALLOWED_AREAS' ? 'bg-success' : 'bg-danger'}">
                                                    ${serviceAreaRestriction.restrictionType || 'N/A'}
                                                </span>
                                            </div>
                                            <div class="mb-2">
                                                <strong>Max TAs:</strong>
                                                <code>${serviceAreaRestriction.maxNumOfTAs || 'N/A'}</code>
                                            </div>
                                            <div class="mb-2">
                                                <strong>Areas:</strong>
                                                ${serviceAreaRestriction.areas && serviceAreaRestriction.areas.length > 0 ? 
                                                    serviceAreaRestriction.areas.map(area => `
                                                        <div class="mt-2 p-2 border rounded">
                                                            <div><strong>Area Code:</strong> <code>${area.areaCodes || 'N/A'}</code></div>
                                                            <div><strong>TACs:</strong> 
                                                                ${area.tacs && area.tacs.length > 0 ? 
                                                                    area.tacs.map(tac => 
                                                                        `<span class="badge bg-info me-1">${tac}</span>`
                                                                    ).join('') : 
                                                                    'No TACs defined'
                                                                }
                                                            </div>
                                                        </div>
                                                    `).join('') : 
                                                    '<div class="text-muted">No areas defined</div>'
                                                }
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            <!-- Additional Settings -->
                            <div class="row">
                                <div class="col-md-6">
                                    <h6 class="border-bottom pb-2"><i class="fas fa-sliders-h me-2"></i>Timers & Settings</h6>
                                    <div class="card bg-light">
                                        <div class="card-body">
                                            <div class="mb-2">
                                                <strong>RFSP Index:</strong>
                                                <code>${amData.rfspIndex !== undefined ? amData.rfspIndex : 'N/A'}</code>
                                            </div>
                                            <div class="mb-2">
                                                <strong>Subscription Registration Timer:</strong>
                                                <code>${amData.subsRegTimer !== undefined ? amData.subsRegTimer : 'N/A'}</code>
                                            </div>
                                            <div class="mb-2">
                                                <strong>UE Usage Type:</strong>
                                                <code>${amData.ueUsageType !== undefined ? amData.ueUsageType : 'N/A'}</code>
                                            </div>
                                            <div class="mb-2">
                                                <strong>Active Time:</strong>
                                                <code>${amData.activeTime !== undefined ? amData.activeTime : 'N/A'}</code>
                                            </div>
                                            <div class="mb-2">
                                                <strong>DL Packet Count:</strong>
                                                <code>${amData.dlPacketCount !== undefined ? amData.dlPacketCount : 'N/A'}</code>
                                            </div>
                                        </div>
                                    </div>
                                </div>

                                <div class="col-md-6">
                                    <h6 class="border-bottom pb-2"><i class="fas fa-flag me-2"></i>Priority & Flags</h6>
                                    <div class="card bg-light">
                                        <div class="card-body">
                                            <div class="mb-2">
                                                <strong>MPS Priority:</strong>
                                                <span class="badge ${amData.mpsPriority ? 'bg-success' : 'bg-secondary'}">
                                                    ${amData.mpsPriority ? 'Enabled' : 'Disabled'}
                                                </span>
                                            </div>
                                            <div class="mb-2">
                                                <strong>MCS Priority:</strong>
                                                <span class="badge ${amData.mcsPriority ? 'bg-success' : 'bg-secondary'}">
                                                    ${amData.mcsPriority ? 'Enabled' : 'Disabled'}
                                                </span>
                                            </div>
                                            <div class="mb-2">
                                                <strong>MICO Allowed:</strong>
                                                <span class="badge ${amData.micoAllowed ? 'bg-success' : 'bg-secondary'}">
                                                    ${amData.micoAllowed ? 'Allowed' : 'Not Allowed'}
                                                </span>
                                            </div>
                                            <div class="mb-2">
                                                <strong>ODB Packet Services:</strong>
                                                <div class="mt-1">
                                                    <span class="badge bg-info">
                                                        ${amData.odbPacketServices || 'N/A'}
                                                    </span>
                                                </div>
                                            </div>
                                            <div class="mb-2">
                                                <strong>Shared AM Data IDs:</strong>
                                                <div class="mt-1">
                                                    ${amData.sharedAmDataIds && amData.sharedAmDataIds.length > 0 ? 
                                                        amData.sharedAmDataIds.map(id => 
                                                            `<span class="badge bg-secondary me-1">${id}</span>`
                                                        ).join('') : 
                                                        '<span class="text-muted">No shared AM data IDs</span>'
                                                    }
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            <!-- SOR Info -->
                            <div class="row mt-4">
                                <div class="col-12">
                                    <h6 class="border-bottom pb-2"><i class="fas fa-sync me-2"></i>Steering of Roaming Information</h6>
                                    <div class="card bg-light">
                                        <div class="card-body">
                                            <div class="row">
                                                <div class="col-md-3">
                                                    <strong>Acknowledgment:</strong>
                                                    <div class="mt-1">
                                                        <span class="badge ${sorInfo.ackInd ? 'bg-success' : 'bg-secondary'}">
                                                            ${sorInfo.ackInd ? 'Required' : 'Not Required'}
                                                        </span>
                                                    </div>
                                                </div>
                                                <div class="col-md-3">
                                                    <strong>MAC IAUSF:</strong>
                                                    <div class="mt-1">
                                                        <code>${sorInfo.sorMacIausf || 'N/A'}</code>
                                                    </div>
                                                </div>
                                                <div class="col-md-3">
                                                    <strong>Counter:</strong>
                                                    <div class="mt-1">
                                                        <code>${sorInfo.countersor || 'N/A'}</code>
                                                    </div>
                                                </div>
                                                <div class="col-md-3">
                                                    <strong>Provisioning Time:</strong>
                                                    <div class="mt-1">
                                                        <code>${sorInfo.provisioningTime || 'N/A'}</code>
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
            </div>

            <!-- SMF Selection Section -->
            <div class="row mt-4">
                <div class="col-12">
                    <div class="card mb-3">
                        <div class="card-header bg-success text-white">
                            <h5 class="mb-0"><i class="fas fa-route me-2"></i>SMF Selection</h5>
                        </div>
                        <div class="card-body">
                            <!-- Supported Features -->
                            <div class="row mb-3">
                                <div class="col-12">
                                    <h6 class="border-bottom pb-2">
                                        <i class="fas fa-star me-2"></i>Supported Features
                                    </h6>
                                    <code>${subscriberData.SmfSelectionSubscriptionData?.supportedFeatures || 'N/A'}</code>
                                </div>
                            </div>

                            <!-- Subscribed NSSAI Info -->
                            <div class="row">
                                <div class="col-12">
                                    <h6 class="border-bottom pb-2">
                                        <i class="fas fa-network-wired me-2"></i>Subscribed NSSAI Information
                                    </h6>
                                    ${Object.entries(subscriberData.SmfSelectionSubscriptionData?.subscribedSnssaiInfos || {}).map(([key, info]) => `
                                        <div class="card bg-light mb-3">
                                            <div class="card-body">
                                                <h6 class="card-subtitle mb-2">NSSAI: ${key}</h6>
                                                ${info.dnnInfos?.map(dnnInfo => `
                                                    <div class="mb-3">
                                                        <strong>DNN:</strong> <code>${dnnInfo.dnn || 'N/A'}</code>
                                                        <div class="mt-2">
                                                            <span class="badge ${dnnInfo.defaultDnnIndicator ? 'bg-success' : 'bg-secondary'} me-2">
                                                                ${dnnInfo.defaultDnnIndicator ? 'Default DNN' : 'Not Default'}
                                                            </span>
                                                            <span class="badge ${dnnInfo.lboRoamingAllowed ? 'bg-success' : 'bg-secondary'} me-2">
                                                                ${dnnInfo.lboRoamingAllowed ? 'LBO Roaming Allowed' : 'LBO Roaming Not Allowed'}
                                                            </span>
                                                            <span class="badge ${dnnInfo.iwkEpsInd ? 'bg-success' : 'bg-secondary'}">
                                                                ${dnnInfo.iwkEpsInd ? 'IWK EPS Enabled' : 'IWK EPS Disabled'}
                                                            </span>
                                                        </div>
                                                    </div>
                                                `).join('') || 'No DNN information available'}
                                            </div>
                                        </div>
                                    `).join('') || 'No NSSAI information available'}
                                </div>
                            </div>

                            <!-- Shared NSSAI Info ID -->
                            <div class="row">
                                <div class="col-12">
                                    <h6 class="border-bottom pb-2">
                                        <i class="fas fa-share-alt me-2"></i>Shared NSSAI Info ID
                                    </h6>
                                    <code>${subscriberData.SmfSelectionSubscriptionData?.sharedSnssaiInfosId || 'N/A'}</code>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Policy Section -->
            <div class="row mt-4">
                <div class="col-12">
                    <div class="card mb-3">
                        <div class="card-header bg-warning text-dark">
                            <h5 class="mb-0"><i class="fas fa-clipboard-list me-2"></i>Policy Information</h5>
                        </div>
                        <div class="card-body">
                            <!-- AM Policy Data -->
                            <div class="row mb-4">
                                <div class="col-12">
                                    <h6 class="border-bottom pb-2">
                                        <i class="fas fa-mobile-alt me-2"></i>Access and Mobility Policy
                                    </h6>
                                    <div class="card bg-light">
                                        <div class="card-body">
                                            <strong>Subscription Categories:</strong><br>
                                            ${subscriberData.AmPolicyData?.subscCats?.map(cat => 
                                                `<span class="badge bg-info me-2">${cat}</span>`
                                            ).join('') || 'No subscription categories defined'}
                                        </div>
                                    </div>
                                </div>
                            </div>

                            <!-- SM Policy Data -->
                            <div class="row">
                                <div class="col-12">
                                    <h6 class="border-bottom pb-2">
                                        <i class="fas fa-server me-2"></i>Session Management Policy
                                    </h6>
                                    ${Object.entries(subscriberData.SmPolicyData?.smPolicySnssaiData || {}).map(([key, data]) => `
                                        <div class="card bg-light mb-3">
                                            <div class="card-header">
                                                <h6 class="mb-0">SNSSAI: SST ${data.snssai?.sst || 'N/A'} ${data.snssai?.sd ? `/ SD ${data.snssai.sd}` : ''}</h6>
                                            </div>
                                            <div class="card-body">
                                                ${Object.entries(data.smPolicyDnnData || {}).map(([dnn, dnnData]) => `
                                                    <div class="mb-3">
                                                        <h6 class="border-bottom pb-2">DNN: ${dnn}</h6>
                                                        <div class="row">
                                                            <!-- Services and Categories -->
                                                            <div class="col-md-6 mb-3">
                                                                <strong>Allowed Services:</strong><br>
                                                                ${dnnData.allowedServices?.map(service => 
                                                                    `<span class="badge bg-info me-1">${service}</span>`
                                                                ).join('') || 'N/A'}<br>
                                                                <strong>Subscription Categories:</strong><br>
                                                                ${dnnData.subscCats?.map(cat => 
                                                                    `<span class="badge bg-secondary me-1">${cat}</span>`
                                                                ).join('') || 'N/A'}
                                                            </div>
                                                            <!-- GBR Values -->
                                                            <div class="col-md-6 mb-3">
                                                                <strong>GBR UL:</strong> <code>${dnnData.gbrUl || 'N/A'}</code><br>
                                                                <strong>GBR DL:</strong> <code>${dnnData.gbrDl || 'N/A'}</code>
                                                            </div>
                                                            <!-- Feature Flags -->
                                                            <div class="col-12 mb-3">
                                                                <div class="d-flex flex-wrap gap-2">
                                                                    <span class="badge ${dnnData.adcSupport ? 'bg-success' : 'bg-secondary'}">
                                                                        ADC Support: ${dnnData.adcSupport ? 'Yes' : 'No'}
                                                                    </span>
                                                                    <span class="badge ${dnnData.subscSpendingLimits ? 'bg-success' : 'bg-secondary'}">
                                                                        Spending Limits: ${dnnData.subscSpendingLimits ? 'Yes' : 'No'}
                                                                    </span>
                                                                    <span class="badge ${dnnData.offline ? 'bg-success' : 'bg-secondary'}">
                                                                        Offline: ${dnnData.offline ? 'Yes' : 'No'}
                                                                    </span>
                                                                    <span class="badge ${dnnData.online ? 'bg-success' : 'bg-secondary'}">
                                                                        Online: ${dnnData.online ? 'Yes' : 'No'}
                                                                    </span>
                                                                    <span class="badge ${dnnData.mpsPriority ? 'bg-success' : 'bg-secondary'}">
                                                                        MPS Priority: ${dnnData.mpsPriority ? 'Yes' : 'No'}
                                                                    </span>
                                                                    <span class="badge ${dnnData.imsSignallingPrio ? 'bg-success' : 'bg-secondary'}">
                                                                        IMS Signalling Priority: ${dnnData.imsSignallingPrio ? 'Yes' : 'No'}
                                                                    </span>
                                                                </div>
                                                            </div>
                                                            <!-- IP Indexes -->
                                                            <div class="col-md-6 mb-3">
                                                                <strong>IPv4 Index:</strong> <code>${dnnData.ipv4Index || 'N/A'}</code><br>
                                                                <strong>IPv6 Index:</strong> <code>${dnnData.ipv6Index || 'N/A'}</code>
                                                            </div>
                                                            <!-- MPS Priority Level -->
                                                            <div class="col-md-6 mb-3">
                                                                <strong>MPS Priority Level:</strong> 
                                                                <code>${dnnData.mpsPriorityLevel || 'N/A'}</code>
                                                            </div>
                                                            <!-- CHF Info -->
                                                            ${dnnData.chfInfo ? `
                                                                <div class="col-12 mb-3">
                                                                    <strong>CHF Information:</strong>
                                                                    <div class="mt-2">
                                                                        <strong>Primary CHF:</strong> 
                                                                        <code>${dnnData.chfInfo.primaryChfAddress || 'N/A'}</code><br>
                                                                        <strong>Secondary CHF:</strong> 
                                                                        <code>${dnnData.chfInfo.secondaryChfAddress || 'N/A'}</code>
                                                                    </div>
                                                                </div>
                                                            ` : ''}
                                                            <!-- Reference UM Data Limit IDs -->
                                                            ${Object.entries(dnnData.refUmDataLimitIds || {}).length > 0 ? `
                                                                <div class="col-12">
                                                                    <strong>Reference UM Data Limit IDs:</strong>
                                                                    ${Object.entries(dnnData.refUmDataLimitIds).map(([limitId, data]) => `
                                                                        <div class="mt-2">
                                                                            <code>${limitId}</code>
                                                                            ${data.monkey?.length > 0 ? `
                                                                                <div class="ms-3">
                                                                                    <small>Monkey IDs:</small><br>
                                                                                    ${data.monkey.map(m => 
                                                                                        `<span class="badge bg-info me-1">${m}</span>`
                                                                                    ).join('')}
                                                                                </div>
                                                                            ` : ''}
                                                                        </div>
                                                                    `).join('')}
                                                                </div>
                                                            ` : ''}
                                                        </div>
                                                    </div>
                                                `).join('')}
                                            </div>
                                        </div>
                                    `).join('')}
                                    
                                    <!-- Usage Monitoring Data -->
                                    <div class="mt-4">
                                        <h6 class="border-bottom pb-2">
                                            <i class="fas fa-chart-line me-2"></i>Usage Monitoring Data
                                        </h6>
                                        
                                        <!-- Usage Monitoring Data Limits -->
                                        ${Object.entries(subscriberData.SmPolicyData?.umDataLimits || {}).map(([limitId, limit]) => `
                                            <div class="card bg-light mb-3">
                                                <div class="card-header">
                                                    <h6 class="mb-0">Limit ID: ${limitId}</h6>
                                                </div>
                                                <div class="card-body">
                                                    <div class="row">
                                                        <div class="col-md-6">
                                                            <strong>Level:</strong> 
                                                            <span class="badge bg-info">${limit.umLevel || 'N/A'}</span>
                                                        </div>
                                                        <div class="col-md-6">
                                                            <strong>Reset Period:</strong> 
                                                            <code>${limit.resetPeriod || 'N/A'}</code>
                                                        </div>
                                                    </div>
                                                    
                                                    <!-- Dates -->
                                                    <div class="row mt-3">
                                                        <div class="col-md-6">
                                                            <strong>Start Date:</strong><br>
                                                            <code>${limit.startDate || 'N/A'}</code>
                                                        </div>
                                                        <div class="col-md-6">
                                                            <strong>End Date:</strong><br>
                                                            <code>${limit.endDate || 'N/A'}</code>
                                                        </div>
                                                    </div>
                                                    
                                                    <!-- Usage Threshold -->
                                                    ${limit.usageLimit ? `
                                                        <div class="row mt-3">
                                                            <div class="col-12">
                                                                <strong>Usage Limits:</strong>
                                                                <div class="mt-2">
                                                                    <div class="d-flex flex-wrap gap-3">
                                                                        ${limit.usageLimit.duration ? `
                                                                            <span>
                                                                                <small class="text-muted">Duration:</small><br>
                                                                                <code>${limit.usageLimit.duration}s</code>
                                                                            </span>
                                                                        ` : ''}
                                                                        ${limit.usageLimit.totalVolume ? `
                                                                            <span>
                                                                                <small class="text-muted">Total Volume:</small><br>
                                                                                <code>${limit.usageLimit.totalVolume} bytes</code>
                                                                            </span>
                                                                        ` : ''}
                                                                        ${limit.usageLimit.downlinkVolume ? `
                                                                            <span>
                                                                                <small class="text-muted">Downlink:</small><br>
                                                                                <code>${limit.usageLimit.downlinkVolume} bytes</code>
                                                                            </span>
                                                                        ` : ''}
                                                                        ${limit.usageLimit.uplinkVolume ? `
                                                                            <span>
                                                                                <small class="text-muted">Uplink:</small><br>
                                                                                <code>${limit.usageLimit.uplinkVolume} bytes</code>
                                                                            </span>
                                                                        ` : ''}
                                                                    </div>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    ` : ''}
                                                    
                                                    <!-- Scopes -->
                                                    ${Object.entries(limit.scopes || {}).length > 0 ? `
                                                        <div class="row mt-3">
                                                            <div class="col-12">
                                                                <strong>Scopes:</strong>
                                                                ${Object.entries(limit.scopes).map(([scopeId, scope]) => `
                                                                    <div class="card bg-white mt-2">
                                                                        <div class="card-body">
                                                                            <strong>Scope ID: ${scopeId}</strong>
                                                                            <div class="mt-2">
                                                                                <strong>SNSSAI:</strong> 
                                                                                SST ${scope.snssai?.sst || 'N/A'}
                                                                                ${scope.snssai?.sd ? `/ SD ${scope.snssai.sd}` : ''}
                                                                            </div>
                                                                            ${scope.dnn?.length > 0 ? `
                                                                                <div class="mt-2">
                                                                                    <strong>DNNs:</strong><br>
                                                                                    ${scope.dnn.map(d => 
                                                                                        `<span class="badge bg-info me-1">${d}</span>`
                                                                                    ).join('')}
                                                                                </div>
                                                                            ` : ''}
                                                                        </div>
                                                                    </div>
                                                                `).join('')}
                                                            </div>
                                                        </div>
                                                    ` : ''}
                                                </div>
                                            </div>
                                        `).join('')}
                                        
                                        <!-- Usage Monitoring Data -->
                                        ${Object.entries(subscriberData.SmPolicyData?.umData || {}).map(([umId, data]) => `
                                            <div class="card bg-light mb-3">
                                                <div class="card-header">
                                                    <h6 class="mb-0">Usage Monitoring ID: ${umId}</h6>
                                                </div>
                                                <div class="card-body">
                                                    <div class="row">
                                                        <div class="col-md-6">
                                                            <strong>Level:</strong> 
                                                            <span class="badge bg-info">${data.umLevel || 'N/A'}</span>
                                                        </div>
                                                        <div class="col-md-6">
                                                            <strong>Reset Time:</strong> 
                                                            <code>${data.resetTime || 'N/A'}</code>
                                                        </div>
                                                    </div>
                                                    
                                                    <!-- Allowed Usage -->
                                                    ${data.allowedUsage ? `
                                                        <div class="row mt-3">
                                                            <div class="col-12">
                                                                <strong>Allowed Usage:</strong>
                                                                <div class="mt-2">
                                                                    <div class="d-flex flex-wrap gap-3">
                                                                        ${data.allowedUsage.duration ? `
                                                                            <span>
                                                                                <small class="text-muted">Duration:</small><br>
                                                                                <code>${data.allowedUsage.duration}s</code>
                                                                            </span>
                                                                        ` : ''}
                                                                        ${data.allowedUsage.totalVolume ? `
                                                                            <span>
                                                                                <small class="text-muted">Total Volume:</small><br>
                                                                                <code>${data.allowedUsage.totalVolume} bytes</code>
                                                                            </span>
                                                                        ` : ''}
                                                                        ${data.allowedUsage.downlinkVolume ? `
                                                                            <span>
                                                                                <small class="text-muted">Downlink:</small><br>
                                                                                <code>${data.allowedUsage.downlinkVolume} bytes</code>
                                                                            </span>
                                                                        ` : ''}
                                                                        ${data.allowedUsage.uplinkVolume ? `
                                                                            <span>
                                                                                <small class="text-muted">Uplink:</small><br>
                                                                                <code>${data.allowedUsage.uplinkVolume} bytes</code>
                                                                            </span>
                                                                        ` : ''}
                                                                    </div>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    ` : ''}
                                                    
                                                    <!-- Scopes -->
                                                    ${Object.entries(data.scopes || {}).length > 0 ? `
                                                        <div class="row mt-3">
                                                            <div class="col-12">
                                                                <strong>Scopes:</strong>
                                                                ${Object.entries(data.scopes).map(([scopeId, scope]) => `
                                                                    <div class="card bg-white mt-2">
                                                                        <div class="card-body">
                                                                            <strong>Scope ID: ${scopeId}</strong>
                                                                            <div class="mt-2">
                                                                                <strong>SNSSAI:</strong> 
                                                                                SST ${scope.snssai?.sst || 'N/A'}
                                                                                ${scope.snssai?.sd ? `/ SD ${scope.snssai.sd}` : ''}
                                                                            </div>
                                                                            ${scope.dnn?.length > 0 ? `
                                                                                <div class="mt-2">
                                                                                    <strong>DNNs:</strong><br>
                                                                                    ${scope.dnn.map(d => 
                                                                                        `<span class="badge bg-info me-1">${d}</span>`
                                                                                    ).join('')}
                                                                                </div>
                                                                            ` : ''}
                                                                        </div>
                                                                    </div>
                                                                `).join('')}
                                                            </div>
                                                        </div>
                                                    ` : ''}
                                                </div>
                                            </div>
                                        `).join('')}
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>`;
    }

    renderEditableDetails(subscriberData) {
        const authData = subscriberData.AuthenticationSubscription || {};
        
        return `
            <form id="subscriberDetailsEditForm">
                <div class="row">
                    <div class="col-md-6">
                        <div class="card mb-3">
                            <div class="card-header">
                                <h6 class="mb-0"><i class="fas fa-edit me-2"></i>Edit Subscriber Information</h6>
                            </div>
                            <div class="card-body">
                                <div class="mb-3">
                                    <label class="form-label">UE ID</label>
                                    <input type="text" class="form-control" id="edit_sub_ueId" 
                                           value="${subscriberData.ueId || ''}" readonly>
                                    <div class="form-text">UE ID cannot be changed</div>
                                </div>
                                <div class="mb-3">
                                    <label class="form-label">PLMN ID</label>
                                    <input type="text" class="form-control" id="edit_sub_plmnID" 
                                           value="${subscriberData.plmnID || ''}" 
                                           placeholder="5 or 6 digits" pattern="\\d{5,6}" maxlength="6" required>
                                </div>
                                <div class="mb-3">
                                    <label class="form-label">Encryption Algorithm</label>
                                    <input type="number" class="form-control" id="edit_sub_encryptionAlgorithm" 
                                           value="${authData.Opc?.EncryptionAlgorithm || 0}" 
                                           placeholder="e.g., 0" min="0">
                                </div>
                                <div class="mb-3">
                                    <label class="form-label">K4 SNO</label>
                                    <select class="form-select" id="edit_sub_k4_sno">
                                        <option value="">Loading K4 keys...</option>
                                    </select>
                                    <div class="form-text">K4 Serial Number reference (optional)</div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="col-md-6">
                        <div class="card mb-3">
                            <div class="card-header">
                                <h6 class="mb-0"><i class="fas fa-key me-2"></i>Authentication Keys</h6>
                            </div>
                            <div class="card-body">
                                <div class="mb-3">
                                    <label class="form-label">Key (Ki)</label>
                                    <input type="text" class="form-control" id="edit_sub_key" 
                                           value="${authData.PermanentKey?.PermanentKeyValue || ''}" 
                                           placeholder="Hexadecimal characters" pattern="[0-9a-fA-F]+" required>
                                </div>
                                <div class="mb-3">
                                    <label class="form-label">OPc</label>
                                    <input type="text" class="form-control" id="edit_sub_opc" 
                                           value="${authData.Opc?.OpcValue || ''}" 
                                           placeholder="Hexadecimal characters" pattern="[0-9a-fA-F]+" required>
                                </div>
                                <div class="mb-3">
                                    <label class="form-label">Sequence Number</label>
                                    <input type="text" class="form-control" id="edit_sub_sequenceNumber" 
                                           value="${authData.SequenceNumber || ''}" 
                                           placeholder="e.g., 16f3b3f70fc2" required>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                
                <div class="row">
                    <div class="col-12">
                        <div class="d-flex justify-content-end">
                            <button type="button" class="btn btn-secondary me-2" onclick="cancelSubscriberEdit()">Cancel</button>
                            <button type="button" class="btn btn-primary" onclick="saveSubscriberDetailsEdit()">Save Changes</button>
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
            await this.updateItem(this.currentSubscriberUeId, payload);
            
            // Refresh the details view
            await this.showDetails(this.currentSubscriberUeId);
            this.toggleEditMode(false);
            
            window.app?.notificationManager?.showNotification('Subscriber updated successfully!', 'success');
            
        } catch (error) {
            console.error('Failed to save subscriber:', error);
            window.app?.notificationManager?.showNotification(`Failed to save subscriber: ${error.message}`, 'error');
        }
    }

    getEditFormData() {
        return {
            sub_ueId: document.getElementById('edit_sub_ueId')?.value || '',
            sub_plmnID: document.getElementById('edit_sub_plmnID')?.value || '',
            sub_key: document.getElementById('edit_sub_key')?.value || '',
            sub_opc: document.getElementById('edit_sub_opc')?.value || '',
            sub_sequenceNumber: document.getElementById('edit_sub_sequenceNumber')?.value || '',
            sub_encryptionAlgorithm: document.getElementById('edit_sub_encryptionAlgorithm')?.value || '',
            sub_k4_sno: document.getElementById('edit_sub_k4_sno')?.value || ''
        };
    }

    toggleEditMode(enable = null) {
        const detailsView = document.getElementById('subscriber-details-view-mode');
        const editView = document.getElementById('subscriber-details-edit-mode');
        const editBtn = document.getElementById('edit-subscriber-btn');
        
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
            
            // Load K4 keys when entering edit mode
            this.loadK4KeysForEdit();
        }
    }

    async deleteFromDetails() {
        try {
            await this.deleteItem(this.currentSubscriberUeId);
            window.app?.notificationManager?.showNotification('Subscriber deleted successfully!', 'success');
            
            // Navigate back to the list
            window.showSection('subscribers-list');
            
        } catch (error) {
            console.error('Failed to delete subscriber:', error);
            window.app?.notificationManager?.showNotification(`Failed to delete subscriber: ${error.message}`, 'error');
        }
    }

    async createFromForm() {
        try {
            const formData = this.getCreateFormData();
            console.log('Form data collected:', formData);
            
            const validation = this.validateFormData(formData);
            
            if (!validation.isValid) {
                console.log('Validation errors:', validation.errors);
                window.app?.notificationManager?.showNotification(validation.errors.join('<br>'), 'error');
                return;
            }

            const payload = this.preparePayload(formData, false);
            console.log('Payload prepared:', payload);
            console.log('UE ID for API call:', formData.sub_ueId);
            
            await this.createItem(payload, formData.sub_ueId);
            
            window.app?.notificationManager?.showNotification('Subscriber created successfully!', 'success');
            
            // Navigate back to the list
            window.showSection('subscribers-list');
            
        } catch (error) {
            console.error('Failed to create subscriber:', error);
            window.app?.notificationManager?.showNotification(`Failed to create subscriber: ${error.message}`, 'error');
        }
    }

    getCreateFormData() {
        return {
            sub_ueId: document.getElementById('sub_ueId')?.value || '',
            sub_plmnID: document.getElementById('sub_plmnID')?.value || '',
            sub_key: document.getElementById('sub_key')?.value || '',
            sub_opc: document.getElementById('sub_opc')?.value || '',
            sub_sequenceNumber: document.getElementById('sub_sequenceNumber')?.value || '',
            sub_encryptionAlgorithm: document.getElementById('sub_encryptionAlgorithm')?.value || '',
            sub_k4_sno: document.getElementById('sub_k4_sno')?.value || ''
        };
    }

    async showCreateForm() {
        // Call parent method first
        await super.showCreateForm();
        
        // Load K4 keys for the dropdown
        await this.loadK4Keys();
    }

    async showEditForm(ueId) {
        // Call parent method first
        await super.showEditForm(ueId);
        
        // Load K4 keys for the dropdown
        await this.loadK4Keys();
    }
}