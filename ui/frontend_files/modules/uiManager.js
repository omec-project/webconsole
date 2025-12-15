// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Canonical Ltd.

import app from '../app.js';

export class UIManager {
    constructor() {
        this.sections = {
            'device-groups': 'deviceGroups',
            'device-group-details': 'deviceGroups', // Same manager for details
            'network-slices': 'networkSlices', 
            'network-slice-details': 'networkSlices', // Same manager for details
            'gnb-inventory': 'gnbInventory',
            'gnb-details': 'gnbInventory', // Same manager for details
            'upf-inventory': 'upfInventory',
            'subscribers': 'subscribers', // identificador genérico
            'k4-keys': 'k4Manager',
            'k4-details': 'k4Manager',
            'subscribers-list': 'subscriberListManager',
            'subscriber-details': 'subscriberListManager'
        };
    }

    showSection(section) {
        // Hide all sections
        document.querySelectorAll('.content-section').forEach(el => {
            el.style.display = 'none';
        });
        
        // Remove active class from all nav links
        document.querySelectorAll('.nav-link').forEach(el => {
            el.classList.remove('active');
        });
        
        // Show selected section
        const sectionElement = document.getElementById(section);
        if (sectionElement) {
            sectionElement.style.display = 'block';
        }
        
        // Add active class to the corresponding nav link by finding it, 
        // instead of relying on event.target.
        // El atributo 'onclick' contiene el nombre de la sección, así que lo usamos como selector.
        const navLink = document.querySelector(`.nav-link[onclick="showSection('${section}')"]`);
        if (navLink) {
            navLink.classList.add('active');
        }
        
        // Update app state
        app.currentSection = section;
        
        // Load data for the section
        this.loadSectionData(section);
    }

    loadSectionData(section) {
        // Lógica para la sección de suscriptores (página original combinada)
        if (section === 'subscribers') {
            app.managers.k4Manager.loadData();
            app.managers.subscriberManager.renderForm();
        } else if (section === 'k4-keys') {
            // Load K4 keys list
            app.managers.k4Manager.loadData();
        } else if (section === 'subscribers-list') {
            // Load subscribers list
            app.managers.subscriberListManager.loadData();
        } else if (section === 'device-groups') {
            // Load device groups list
            const managerKey = this.sections[section];
            if (managerKey && app.managers[managerKey]) {
                app.managers[managerKey].loadData();
            }
        } else if (section === 'network-slices') {
            // Load network slices list
            const managerKey = this.sections[section];
            if (managerKey && app.managers[managerKey]) {
                app.managers[managerKey].loadData();
            }
        } else if (section === 'gnb-inventory') {
            // Load gNB inventory list
            const managerKey = this.sections[section];
            if (managerKey && app.managers[managerKey]) {
                app.managers[managerKey].loadData();
            }
        } else if (section === 'device-group-details' || section === 'gnb-details' || section === 'network-slice-details' || section === 'k4-details' || section === 'subscriber-details') {
            // Don't reload data for details views as they're already loaded
            return;
        } else {
            const managerKey = this.sections[section];
            if (managerKey && app.managers[managerKey]) {
                app.managers[managerKey].loadData();
            }
        }
    }

    showLoading(containerId) {
        const container = document.getElementById(containerId);
        if (container) {
            container.innerHTML = `
                <div class="text-center p-4">
                    <div class="spinner-border" role="status">
                        <span class="visually-hidden">Loading...</span>
                    </div>
                    <p class="mt-2">Loading data...</p>
                </div>
            `;
        }
    }

    showError(containerId, message) {
        const container = document.getElementById(containerId);
        if (container) {
            container.innerHTML = `
                <div class="alert alert-danger">
                    <i class="fas fa-exclamation-triangle me-2"></i>
                    ${message}
                </div>
            `;
        }
    }

    showEmpty(containerId, message) {
        const container = document.getElementById(containerId);
        if (container) {
            container.innerHTML = `
                <div class="alert alert-info">
                    <i class="fas fa-info-circle me-2"></i>
                    ${message}
                </div>
            `;
        }
    }
}
