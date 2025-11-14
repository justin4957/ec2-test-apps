/**
 * Offline Storage Module for Solid PoC
 *
 * Provides:
 * - IndexedDB for local data caching
 * - Operation queue for offline writes
 * - Sync logic for pending operations
 * - Conflict resolution strategies
 */

class OfflineStorage {
    constructor() {
        this.dbName = 'SolidPoCDB';
        this.dbVersion = 1;
        this.db = null;
        this.syncCallbacks = [];
    }

    /**
     * Initialize IndexedDB
     */
    async init() {
        return new Promise((resolve, reject) => {
            const request = indexedDB.open(this.dbName, this.dbVersion);

            request.onerror = () => {
                console.error('[OfflineStorage] Failed to open IndexedDB:', request.error);
                reject(request.error);
            };

            request.onsuccess = () => {
                this.db = request.result;
                console.log('[OfflineStorage] IndexedDB initialized');
                resolve(this.db);
            };

            request.onupgradeneeded = (event) => {
                const db = event.target.result;
                console.log('[OfflineStorage] Upgrading database schema');

                // Store for cached location data
                if (!db.objectStoreNames.contains('locations')) {
                    const locationStore = db.createObjectStore('locations', {
                        keyPath: 'id',
                        autoIncrement: true
                    });
                    locationStore.createIndex('url', 'url', { unique: true });
                    locationStore.createIndex('timestamp', 'timestamp', { unique: false });
                    locationStore.createIndex('synced', 'synced', { unique: false });
                }

                // Store for pending operations (write queue)
                if (!db.objectStoreNames.contains('pendingOps')) {
                    const opsStore = db.createObjectStore('pendingOps', {
                        keyPath: 'id',
                        autoIncrement: true
                    });
                    opsStore.createIndex('timestamp', 'timestamp', { unique: false });
                    opsStore.createIndex('retryCount', 'retryCount', { unique: false });
                    opsStore.createIndex('status', 'status', { unique: false });
                }

                // Store for cached profile data
                if (!db.objectStoreNames.contains('profiles')) {
                    const profileStore = db.createObjectStore('profiles', {
                        keyPath: 'webId'
                    });
                    profileStore.createIndex('lastUpdated', 'lastUpdated', { unique: false });
                }

                // Store for sync metadata
                if (!db.objectStoreNames.contains('syncMeta')) {
                    db.createObjectStore('syncMeta', { keyPath: 'key' });
                }
            };
        });
    }

    /**
     * Cache location data locally
     */
    async cacheLocation(url, data, synced = true) {
        if (!this.db) {
            throw new Error('Database not initialized');
        }

        const transaction = this.db.transaction(['locations'], 'readwrite');
        const store = transaction.objectStore('locations');

        const locationData = {
            url,
            data,
            timestamp: Date.now(),
            synced,
            lastModified: new Date().toISOString()
        };

        return new Promise((resolve, reject) => {
            // First try to get existing entry by URL
            const urlIndex = store.index('url');
            const getRequest = urlIndex.get(url);

            getRequest.onsuccess = () => {
                const existingData = getRequest.result;

                if (existingData) {
                    // Update existing entry
                    const updateData = { ...locationData, id: existingData.id };
                    const updateRequest = store.put(updateData);

                    updateRequest.onsuccess = () => {
                        console.log('[OfflineStorage] Location updated in cache:', url);
                        resolve(updateRequest.result);
                    };

                    updateRequest.onerror = () => reject(updateRequest.error);
                } else {
                    // Add new entry
                    const addRequest = store.add(locationData);

                    addRequest.onsuccess = () => {
                        console.log('[OfflineStorage] Location cached:', url);
                        resolve(addRequest.result);
                    };

                    addRequest.onerror = () => reject(addRequest.error);
                }
            };

            getRequest.onerror = () => reject(getRequest.error);
        });
    }

    /**
     * Get cached location data
     */
    async getCachedLocation(url) {
        if (!this.db) {
            throw new Error('Database not initialized');
        }

        const transaction = this.db.transaction(['locations'], 'readonly');
        const store = transaction.objectStore('locations');
        const index = store.index('url');

        return new Promise((resolve, reject) => {
            const request = index.get(url);

            request.onsuccess = () => {
                if (request.result) {
                    console.log('[OfflineStorage] Location found in cache:', url);
                    resolve(request.result);
                } else {
                    resolve(null);
                }
            };

            request.onerror = () => reject(request.error);
        });
    }

    /**
     * Get all cached locations
     */
    async getAllCachedLocations() {
        if (!this.db) {
            throw new Error('Database not initialized');
        }

        const transaction = this.db.transaction(['locations'], 'readonly');
        const store = transaction.objectStore('locations');

        return new Promise((resolve, reject) => {
            const request = store.getAll();

            request.onsuccess = () => {
                console.log('[OfflineStorage] Retrieved all cached locations:', request.result.length);
                resolve(request.result);
            };

            request.onerror = () => reject(request.error);
        });
    }

    /**
     * Queue a Pod operation for later sync
     */
    async queueOperation(operation) {
        if (!this.db) {
            throw new Error('Database not initialized');
        }

        const transaction = this.db.transaction(['pendingOps'], 'readwrite');
        const store = transaction.objectStore('pendingOps');

        const opData = {
            ...operation,
            timestamp: Date.now(),
            retryCount: 0,
            status: 'pending', // pending, syncing, failed, completed
            lastAttempt: null,
            error: null
        };

        return new Promise((resolve, reject) => {
            const request = store.add(opData);

            request.onsuccess = () => {
                console.log('[OfflineStorage] Operation queued:', operation.type);
                resolve(request.result);
            };

            request.onerror = () => reject(request.error);
        });
    }

    /**
     * Get all pending operations
     */
    async getPendingOperations() {
        if (!this.db) {
            throw new Error('Database not initialized');
        }

        const transaction = this.db.transaction(['pendingOps'], 'readonly');
        const store = transaction.objectStore('pendingOps');
        const index = store.index('status');

        return new Promise((resolve, reject) => {
            const request = index.getAll('pending');

            request.onsuccess = () => {
                console.log('[OfflineStorage] Retrieved pending operations:', request.result.length);
                resolve(request.result);
            };

            request.onerror = () => reject(request.error);
        });
    }

    /**
     * Update operation status
     */
    async updateOperationStatus(id, status, error = null) {
        if (!this.db) {
            throw new Error('Database not initialized');
        }

        const transaction = this.db.transaction(['pendingOps'], 'readwrite');
        const store = transaction.objectStore('pendingOps');

        return new Promise((resolve, reject) => {
            const getRequest = store.get(id);

            getRequest.onsuccess = () => {
                const op = getRequest.result;

                if (!op) {
                    reject(new Error(`Operation ${id} not found`));
                    return;
                }

                op.status = status;
                op.lastAttempt = Date.now();

                if (error) {
                    op.error = error;
                    op.retryCount = (op.retryCount || 0) + 1;
                }

                const updateRequest = store.put(op);

                updateRequest.onsuccess = () => {
                    console.log(`[OfflineStorage] Operation ${id} status updated to ${status}`);
                    resolve(updateRequest.result);
                };

                updateRequest.onerror = () => reject(updateRequest.error);
            };

            getRequest.onerror = () => reject(getRequest.error);
        });
    }

    /**
     * Delete completed operation
     */
    async deleteOperation(id) {
        if (!this.db) {
            throw new Error('Database not initialized');
        }

        const transaction = this.db.transaction(['pendingOps'], 'readwrite');
        const store = transaction.objectStore('pendingOps');

        return new Promise((resolve, reject) => {
            const request = store.delete(id);

            request.onsuccess = () => {
                console.log('[OfflineStorage] Operation deleted:', id);
                resolve();
            };

            request.onerror = () => reject(request.error);
        });
    }

    /**
     * Cache profile data
     */
    async cacheProfile(webId, profileData) {
        if (!this.db) {
            throw new Error('Database not initialized');
        }

        const transaction = this.db.transaction(['profiles'], 'readwrite');
        const store = transaction.objectStore('profiles');

        const data = {
            webId,
            profileData,
            lastUpdated: Date.now()
        };

        return new Promise((resolve, reject) => {
            const request = store.put(data);

            request.onsuccess = () => {
                console.log('[OfflineStorage] Profile cached:', webId);
                resolve(request.result);
            };

            request.onerror = () => reject(request.error);
        });
    }

    /**
     * Get cached profile
     */
    async getCachedProfile(webId) {
        if (!this.db) {
            throw new Error('Database not initialized');
        }

        const transaction = this.db.transaction(['profiles'], 'readonly');
        const store = transaction.objectStore('profiles');

        return new Promise((resolve, reject) => {
            const request = store.get(webId);

            request.onsuccess = () => {
                if (request.result) {
                    console.log('[OfflineStorage] Profile found in cache:', webId);
                    resolve(request.result);
                } else {
                    resolve(null);
                }
            };

            request.onerror = () => reject(request.error);
        });
    }

    /**
     * Get/set sync metadata
     */
    async getSyncMeta(key) {
        if (!this.db) {
            throw new Error('Database not initialized');
        }

        const transaction = this.db.transaction(['syncMeta'], 'readonly');
        const store = transaction.objectStore('syncMeta');

        return new Promise((resolve, reject) => {
            const request = store.get(key);

            request.onsuccess = () => {
                resolve(request.result ? request.result.value : null);
            };

            request.onerror = () => reject(request.error);
        });
    }

    async setSyncMeta(key, value) {
        if (!this.db) {
            throw new Error('Database not initialized');
        }

        const transaction = this.db.transaction(['syncMeta'], 'readwrite');
        const store = transaction.objectStore('syncMeta');

        return new Promise((resolve, reject) => {
            const request = store.put({ key, value });

            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    }

    /**
     * Sync pending operations with Pod
     */
    async syncPendingOperations(executeOperationFn) {
        const pendingOps = await this.getPendingOperations();

        if (pendingOps.length === 0) {
            console.log('[OfflineStorage] No pending operations to sync');
            return { succeeded: 0, failed: 0 };
        }

        console.log(`[OfflineStorage] Syncing ${pendingOps.length} pending operations`);

        let succeeded = 0;
        let failed = 0;

        for (const op of pendingOps) {
            try {
                // Update status to syncing
                await this.updateOperationStatus(op.id, 'syncing');

                // Execute the operation
                await executeOperationFn(op);

                // Delete successful operation
                await this.deleteOperation(op.id);

                succeeded++;
                console.log(`[OfflineStorage] Operation ${op.id} synced successfully`);
            } catch (error) {
                console.error(`[OfflineStorage] Operation ${op.id} failed:`, error);

                // Mark as failed with error
                await this.updateOperationStatus(op.id, 'pending', error.message);

                failed++;

                // If retry count exceeds threshold, mark as permanently failed
                if (op.retryCount >= 5) {
                    await this.updateOperationStatus(op.id, 'failed', error.message);
                }
            }
        }

        console.log(`[OfflineStorage] Sync complete - ${succeeded} succeeded, ${failed} failed`);

        // Update last sync timestamp
        await this.setSyncMeta('lastSync', Date.now());

        return { succeeded, failed };
    }

    /**
     * Clear all data (for testing/reset)
     */
    async clearAll() {
        if (!this.db) {
            throw new Error('Database not initialized');
        }

        const storeNames = ['locations', 'pendingOps', 'profiles', 'syncMeta'];
        const transaction = this.db.transaction(storeNames, 'readwrite');

        const promises = storeNames.map(storeName => {
            return new Promise((resolve, reject) => {
                const request = transaction.objectStore(storeName).clear();
                request.onsuccess = () => resolve();
                request.onerror = () => reject(request.error);
            });
        });

        await Promise.all(promises);
        console.log('[OfflineStorage] All data cleared');
    }

    /**
     * Get sync statistics
     */
    async getSyncStats() {
        const transaction = this.db.transaction(['pendingOps', 'locations'], 'readonly');

        const opsStore = transaction.objectStore('pendingOps');
        const locationsStore = transaction.objectStore('locations');

        const [allOps, allLocations] = await Promise.all([
            new Promise(resolve => {
                const req = opsStore.getAll();
                req.onsuccess = () => resolve(req.result);
            }),
            new Promise(resolve => {
                const req = locationsStore.getAll();
                req.onsuccess = () => resolve(req.result);
            })
        ]);

        const pendingCount = allOps.filter(op => op.status === 'pending').length;
        const failedCount = allOps.filter(op => op.status === 'failed').length;
        const syncedLocations = allLocations.filter(loc => loc.synced).length;
        const unsyncedLocations = allLocations.filter(loc => !loc.synced).length;

        const lastSync = await this.getSyncMeta('lastSync');

        return {
            pendingOperations: pendingCount,
            failedOperations: failedCount,
            cachedLocations: allLocations.length,
            syncedLocations,
            unsyncedLocations,
            lastSync: lastSync ? new Date(lastSync).toISOString() : 'Never'
        };
    }
}

// Create singleton instance
const offlineStorage = new OfflineStorage();

// Export for use in window context
if (typeof window !== 'undefined') {
    window.offlineStorage = offlineStorage;
}
