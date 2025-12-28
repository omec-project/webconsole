// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

export class NotificationManager {
    constructor() {
        this.toastElement = null;
        this.toastBodyElement = null;
        this.initializeElements();
    }

    initializeElements() {
        this.toastElement = document.getElementById('notificationToast');
        this.toastBodyElement = document.getElementById('toastBody');
    }

    showSuccess(message) {
        this.showNotification(message, 'success');
    }

    showError(message) {
        this.showNotification(message, 'danger');
    }

    showWarning(message) {
        this.showNotification(message, 'warning');
    }

    showInfo(message) {
        this.showNotification(message, 'info');
    }

    showNotification(message, type) {
        if (!this.toastElement || !this.toastBodyElement) {
            console.error('Toast elements not found');
            return;
        }

        // Set toast styling based on type
        const typeClasses = {
            success: 'bg-success text-white',
            danger: 'bg-danger text-white', 
            warning: 'bg-warning text-dark',
            info: 'bg-info text-white'
        };

        this.toastElement.className = `toast ${typeClasses[type] || typeClasses.info}`;
        this.toastBodyElement.textContent = message;
        
        // Show the toast
        const bsToast = new bootstrap.Toast(this.toastElement, {
            autohide: true,
            delay: type === 'error' ? 5000 : 3000
        });
        bsToast.show();
    }

    showApiError(error, operation = 'operation') {
        let message = `Failed to ${operation}`;
        
        if (error.message) {
            message += `: ${error.message}`;
        } else if (typeof error === 'string') {
            message += `: ${error}`;
        }
        
        this.showError(message);
    }
}
