// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

export class ModalManager {
    constructor() {
        this.currentEditType = '';
        this.currentEditName = '';
        this.modal = null;
        this.initializeModal();
    }

    initializeModal() {
        const modalElement = document.getElementById('createEditModal');
        if (modalElement) {
            this.modal = new bootstrap.Modal(modalElement);
        }
    }

    async showCreateForm(type) {
        this.currentEditType = type;
        this.currentEditName = '';
        
        const manager = this.getManagerByType(type);
        if (!manager) {
            window.app.notificationManager.showError(`Unknown type: ${type}`);
            return;
        }
        
        document.getElementById('modalTitle').textContent = `Create ${manager.displayName}`;
        this.generateForm(manager, false);
        
        // Call manager's showCreateForm if it exists
        if (typeof manager.showCreateForm === 'function') {
            await manager.showCreateForm();
        }
        
        this.modal.show();
    }

    async editItem(type, name) {
        this.currentEditType = type;
        this.currentEditName = name;
        
        const manager = this.getManagerByType(type);
        if (!manager) {
            window.app.notificationManager.showError(`Unknown type: ${type}`);
            return;
        }
        
        document.getElementById('modalTitle').textContent = `Edit ${manager.displayName}: ${name}`;
        this.generateForm(manager, true);
        
        // Call manager's showEditForm if it exists, otherwise use default loadItemData
        if (typeof manager.showEditForm === 'function') {
            await manager.showEditForm(name);
        } else {
            await this.loadItemData(manager, name);
        }
        
        this.modal.show();
    }

    async deleteItem(type, name) {
        const manager = this.getManagerByType(type);
        if (!manager) {
            app.notificationManager.showError(`Unknown type: ${type}`);
            return;
        }

        const confirmed = confirm(`Are you sure you want to delete ${manager.displayName}: ${name}?`);
        if (!confirmed) return;

        try {
            await manager.deleteItem(name);
            app.notificationManager.showSuccess(`${manager.displayName} deleted successfully`);
            manager.loadData(); // Reload the list
        } catch (error) {
            app.notificationManager.showApiError(error, 'delete item');
        }
    }

    async saveItem() {
        const manager = this.getManagerByType(this.currentEditType);
        if (!manager) {
            window.app.notificationManager.showError(`Unknown type: ${this.currentEditType}`);
            return;
        }

        // Collect form data
        const formData = this.collectFormData();
        
        // Validate form data
        const validation = manager.validateFormData(formData);
        if (!validation.isValid) {
            const errorMessage = validation.errors.join('\n');
            window.app.notificationManager.showError(errorMessage);
            return;
        }

        try {
            // Prepare payload
            const payload = manager.preparePayload(formData, !!this.currentEditName);
            
            // Save or update
            if (this.currentEditName) {
                await manager.updateItem(this.currentEditName, payload);
                window.app.notificationManager.showSuccess(`${manager.displayName} updated successfully`);
            } else {
                await manager.createItem(payload);
                window.app.notificationManager.showSuccess(`${manager.displayName} created successfully`);
            }

            // Close modal and reload data
            this.modal.hide();
            manager.loadData();

        } catch (error) {
            window.app.notificationManager.showApiError(error, this.currentEditName ? 'update item' : 'create item');
        }
    }

    generateForm(manager, isEdit = false) {
        const container = document.getElementById('formFields');
        const formHtml = manager.getFormFields(isEdit);
        container.innerHTML = formHtml;
    }

    async loadItemData(manager, name) {
        try {
            const data = await manager.getItem(name);
            
            // Populate form fields
            Object.keys(data).forEach(key => {
                const field = document.getElementById(key);
                if (field) {
                    if (field.type === 'checkbox') {
                        field.checked = !!data[key];
                    } else {
                        field.value = data[key] || '';
                    }
                }
            });
        } catch (error) {
            app.notificationManager.showApiError(error, 'load item data');
        }
    }

    collectFormData() {
        const data = {};
        
        // Get all form inputs
        document.querySelectorAll('#formFields input, #formFields textarea, #formFields select').forEach(input => {
            if (input.type === 'checkbox') {
                data[input.id] = input.checked;
            } else if (input.type === 'number') {
                data[input.id] = input.value ? parseInt(input.value) : undefined;
            } else if (input.multiple) {
                // Handle multi-select
                const selectedValues = Array.from(input.selectedOptions).map(option => option.value);
                data[input.id] = selectedValues.filter(value => value); // Remove empty values
            } else {
                data[input.id] = input.value || undefined;
            }
        });
        
        return data;
    }

    getManagerByType(type) {
        const typeMapping = {
            'device-group': 'deviceGroups',
            'network-slice': 'networkSlices',
            'gnb': 'gnbInventory',
            'upf': 'upfInventory',
            'k4-key': 'k4Manager',
            'subscriber': 'subscriberListManager'
        };
        
        const managerKey = typeMapping[type];
        return managerKey ? window.app.managers[managerKey] : null;
    }
}
