// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

// Import modules
import { DeviceGroupManager } from './modules/deviceGroups.js';
import { NetworkSliceManager } from './modules/networkSlices.js';
import { GnbManager } from './modules/gnbInventory.js';
import { UpfManager } from './modules/upfInventory.js';
import { UIManager } from './modules/uiManager.js';
import { NotificationManager } from './modules/notifications.js';
import { ModalManager } from './modules/modalManager.js';
import { SubscriberListManager } from './modules/subscribers.js';
import { K4Manager } from './modules/k4.js';

// API Base URL
export const API_BASE = '/config/v1';
export const SUBSCRIBER_API_BASE = '/api';

// Global application state
class AppState {
    constructor() {
        this.currentSection = 'device-groups';
        this.managers = {
            deviceGroups: new DeviceGroupManager(),
            networkSlices: new NetworkSliceManager(),
            gnbInventory: new GnbManager(),
            upfInventory: new UpfManager(),
            k4Manager: new K4Manager(),
            subscriberListManager: new SubscriberListManager()
        };
        this.uiManager = new UIManager();
        this.notificationManager = new NotificationManager();
        this.modalManager = new ModalManager();
    }

    getCurrentManager() {
        return this.managers[this.currentSection];
    }
}

// Global app instance
const app = new AppState();

// Make app globally accessible
window.app = app;

// Initialize the application
document.addEventListener('DOMContentLoaded', function() {
    app.uiManager.showSection('device-groups');
});

// Export global functions for HTML onclick handlers
window.showSection = (section) => app.uiManager.showSection(section);
window.showCreateForm = async (type) => await app.modalManager.showCreateForm(type);
window.editItem = async (type, name) => await app.modalManager.editItem(type, name);
window.deleteItem = async (type, name) => await app.modalManager.deleteItem(type, name);
window.deleteK4Item = async (k4Sno, keyLabel) => {
    // Special delete handler for K4 keys that requires both sno and key_label
    const confirmed = confirm(`Are you sure you want to delete K4 key with SNO ${k4Sno} and label ${keyLabel}?`);
    if (!confirmed) return;
    
    try {
        await app.managers.k4Manager.deleteItem(k4Sno, keyLabel);
        app.notificationManager.showNotification('K4 key deleted successfully!', 'success');
        await app.managers.k4Manager.loadData();
    } catch (error) {
        console.error('Failed to delete K4 key:', error);
        app.notificationManager.showNotification(`Failed to delete K4 key: ${error.message}`, 'error');
    }
};
window.saveItem = async () => await app.modalManager.saveItem();

// Device Group Details functions
window.showDeviceGroupDetails = async (groupName) => {
    await app.managers.deviceGroups.showDetails(groupName);
    app.uiManager.showSection('device-group-details');
};

window.toggleEditMode = () => {
    app.managers.deviceGroups.toggleEditMode();
};

window.cancelEdit = () => {
    app.managers.deviceGroups.toggleEditMode(false);
};

window.saveDetailsEdit = async () => {
    await app.managers.deviceGroups.saveEdit();
};

window.confirmDeleteDeviceGroup = () => {
    const modal = new bootstrap.Modal(document.getElementById('deleteConfirmModal'));
    document.getElementById('deleteConfirmMessage').textContent = 
        `Are you sure you want to delete the device group "${app.managers.deviceGroups.currentGroupName}"? This action cannot be undone.`;
    
    window.currentDeleteAction = () => app.managers.deviceGroups.deleteFromDetails();
    modal.show();
};

// gNB Details functions
window.showGnbDetails = async (gnbName) => {
    await app.managers.gnbInventory.showDetails(gnbName);
    app.uiManager.showSection('gnb-details');
};

window.toggleGnbEditMode = () => {
    app.managers.gnbInventory.toggleEditMode();
};

window.cancelGnbEdit = () => {
    app.managers.gnbInventory.toggleEditMode(false);
};

window.saveGnbDetailsEdit = async () => {
    await app.managers.gnbInventory.saveEdit();
};

window.confirmDeleteGnb = () => {
    const modal = new bootstrap.Modal(document.getElementById('deleteConfirmModal'));
    document.getElementById('deleteConfirmMessage').textContent = 
        `Are you sure you want to delete the gNB "${app.managers.gnbInventory.currentGnbName}"? This action cannot be undone.`;
    
    window.currentDeleteAction = () => app.managers.gnbInventory.deleteFromDetails();
    modal.show();
};

// Network Slice Details functions
window.showNetworkSliceDetails = async (sliceName) => {
    await app.managers.networkSlices.showDetails(sliceName);
    app.uiManager.showSection('network-slice-details');
};

window.toggleNetworkSliceEditMode = () => {
    app.managers.networkSlices.toggleEditMode();
};

window.cancelNetworkSliceEdit = () => {
    app.managers.networkSlices.toggleEditMode(false);
};

window.saveNetworkSliceDetailsEdit = async () => {
    await app.managers.networkSlices.saveEdit();
};

window.confirmDeleteNetworkSlice = () => {
    const modal = new bootstrap.Modal(document.getElementById('deleteConfirmModal'));
    document.getElementById('deleteConfirmMessage').textContent = 
        `Are you sure you want to delete the network slice "${app.managers.networkSlices.currentSliceName}"? This action cannot be undone.`;
    
    window.currentDeleteAction = () => app.managers.networkSlices.deleteFromDetails();
    modal.show();
};

window.executeDelete = async () => {
    if (window.currentDeleteAction) {
        await window.currentDeleteAction();
        bootstrap.Modal.getInstance(document.getElementById('deleteConfirmModal')).hide();
        window.currentDeleteAction = null;
    }
};

// K4 Details functions
window.showK4Details = async (k4Sno) => {
    await app.managers.k4Manager.showDetails(k4Sno);
    app.uiManager.showSection('k4-details');
};

window.toggleK4EditMode = () => {
    app.managers.k4Manager.toggleEditMode();
};

window.cancelK4Edit = () => {
    app.managers.k4Manager.toggleEditMode(false);
};

window.saveK4DetailsEdit = async () => {
    await app.managers.k4Manager.saveEdit();
};

window.confirmDeleteK4 = () => {
    const modal = new bootstrap.Modal(document.getElementById('deleteConfirmModal'));
    document.getElementById('deleteConfirmMessage').textContent = 
        `Are you sure you want to delete the K4 key "${app.managers.k4Manager.currentK4Sno}"? This action cannot be undone.`;
    
    window.currentDeleteAction = () => app.managers.k4Manager.deleteFromDetails();
    modal.show();
};

// Subscriber Details functions
window.showSubscriberDetails = async (imsi) => {
    await app.managers.subscriberListManager.showDetails(imsi);
    app.uiManager.showSection('subscriber-details');
};

window.toggleSubscriberEditMode = () => {
    app.managers.subscriberListManager.toggleEditMode();
};

window.cancelSubscriberEdit = () => {
    app.managers.subscriberListManager.toggleEditMode(false);
};

window.saveSubscriberDetailsEdit = async () => {
    await app.managers.subscriberListManager.saveEdit();
};

window.confirmDeleteSubscriber = () => {
    const modal = new bootstrap.Modal(document.getElementById('deleteConfirmModal'));
    document.getElementById('deleteConfirmMessage').textContent = 
        `Are you sure you want to delete the subscriber "${app.managers.subscriberListManager.currentSubscriberImsi}"? This action cannot be undone.`;
    
    window.currentDeleteAction = () => app.managers.subscriberListManager.deleteFromDetails();
    modal.show();
};

// Admin Options - SSM Sync functions
window.syncK4Keys = async () => {
    const resultsDiv = document.getElementById('admin-results');
    resultsDiv.innerHTML = '<div class="text-center"><div class="spinner-border text-primary" role="status"></div><p class="mt-2">Executing sync...</p></div>';
    
    try {
        const response = await fetch('/sync-ssm/sync-key');
        const data = await response.text();
        
        if (response.ok) {
            resultsDiv.innerHTML = `
                <div class="alert alert-success mb-0">
                    <h6><i class="fas fa-check-circle me-2"></i>Sync K4 Keys - Success</h6>
                    <p class="mb-0">${data}</p>
                </div>
            `;
            app.notificationManager.showNotification('K4 keys synchronized successfully!', 'success');
        } else {
            throw new Error(data || 'Sync failed');
        }
    } catch (error) {
        resultsDiv.innerHTML = `
            <div class="alert alert-danger mb-0">
                <h6><i class="fas fa-exclamation-circle me-2"></i>Sync K4 Keys - Error</h6>
                <p class="mb-0">${error.message}</p>
            </div>
        `;
        app.notificationManager.showNotification(`Sync failed: ${error.message}`, 'error');
    }
};

window.checkK4Life = async () => {
    const resultsDiv = document.getElementById('admin-results');
    resultsDiv.innerHTML = '<div class="text-center"><div class="spinner-border text-success" role="status"></div><p class="mt-2">Checking K4 life...</p></div>';
    
    try {
        const response = await fetch('/sync-ssm/check-k4-life');
        const data = await response.text();
        
        if (response.ok) {
            resultsDiv.innerHTML = `
                <div class="alert alert-success mb-0">
                    <h6><i class="fas fa-check-circle me-2"></i>Check K4 Life - Success</h6>
                    <p class="mb-0">${data}</p>
                </div>
            `;
            app.notificationManager.showNotification('K4 life check completed successfully!', 'success');
        } else {
            throw new Error(data || 'Health check failed');
        }
    } catch (error) {
        resultsDiv.innerHTML = `
            <div class="alert alert-danger mb-0">
                <h6><i class="fas fa-exclamation-circle me-2"></i>Check K4 Life - Error</h6>
                <p class="mb-0">${error.message}</p>
            </div>
        `;
        app.notificationManager.showNotification(`Health check failed: ${error.message}`, 'error');
    }
};

window.rotateK4Keys = async () => {
    const resultsDiv = document.getElementById('admin-results');
    resultsDiv.innerHTML = '<div class="text-center"><div class="spinner-border text-warning" role="status"></div><p class="mt-2">Executing rotation...</p></div>';
    
    try {
        const response = await fetch('/sync-ssm/k4-rotation');
        const data = await response.text();
        
        if (response.ok) {
            resultsDiv.innerHTML = `
                <div class="alert alert-success mb-0">
                    <h6><i class="fas fa-check-circle me-2"></i>K4 Rotation - Success</h6>
                    <p class="mb-0">${data}</p>
                </div>
            `;
            app.notificationManager.showNotification('K4 rotation executed successfully!', 'success');
        } else {
            throw new Error(data || 'Rotation failed');
        }
    } catch (error) {
        resultsDiv.innerHTML = `
            <div class="alert alert-danger mb-0">
                <h6><i class="fas fa-exclamation-circle me-2"></i>K4 Rotation - Error</h6>
                <p class="mb-0">${error.message}</p>
            </div>
        `;
        app.notificationManager.showNotification(`Rotation failed: ${error.message}`, 'error');
    }
};

// Export app instance for modules
export default app;


