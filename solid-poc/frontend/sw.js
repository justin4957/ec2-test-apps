/**
 * Service Worker for Solid PoC Offline-First Capabilities
 *
 * Provides:
 * - Asset caching for offline use
 * - Network-first strategy for API calls
 * - Background sync for queued operations
 * - Offline/online state management
 */

const CACHE_VERSION = 'solid-poc-v1';
const STATIC_CACHE = `${CACHE_VERSION}-static`;
const DYNAMIC_CACHE = `${CACHE_VERSION}-dynamic`;

// Files to cache immediately on install
const STATIC_ASSETS = [
    '/',
    '/index.html',
    '/dist/solid-client-bundle.js',
    '/offline.js'
];

// Install event - cache static assets
self.addEventListener('install', (event) => {
    console.log('[ServiceWorker] Installing...');

    event.waitUntil(
        caches.open(STATIC_CACHE)
            .then((cache) => {
                console.log('[ServiceWorker] Caching static assets');
                return cache.addAll(STATIC_ASSETS);
            })
            .then(() => {
                console.log('[ServiceWorker] Installation complete');
                // Force the waiting service worker to become the active service worker
                return self.skipWaiting();
            })
            .catch((error) => {
                console.error('[ServiceWorker] Installation failed:', error);
            })
    );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
    console.log('[ServiceWorker] Activating...');

    event.waitUntil(
        caches.keys()
            .then((cacheNames) => {
                return Promise.all(
                    cacheNames
                        .filter((cacheName) => {
                            // Remove old caches
                            return cacheName.startsWith('solid-poc-') &&
                                   cacheName !== STATIC_CACHE &&
                                   cacheName !== DYNAMIC_CACHE;
                        })
                        .map((cacheName) => {
                            console.log('[ServiceWorker] Deleting old cache:', cacheName);
                            return caches.delete(cacheName);
                        })
                );
            })
            .then(() => {
                console.log('[ServiceWorker] Activation complete');
                // Take control of all pages immediately
                return self.clients.claim();
            })
    );
});

// Fetch event - serve from cache or network
self.addEventListener('fetch', (event) => {
    const { request } = event;
    const url = new URL(request.url);

    // Skip non-GET requests
    if (request.method !== 'GET') {
        return;
    }

    // Skip chrome-extension and other non-http(s) requests
    if (!request.url.startsWith('http')) {
        return;
    }

    // Skip Solid Pod requests (always use network for Pod operations)
    if (url.hostname.includes('solidcommunity.net') ||
        url.hostname.includes('inrupt.com') ||
        url.hostname.includes('pod.')) {
        event.respondWith(fetch(request));
        return;
    }

    // Network-first strategy for API calls
    if (url.pathname.startsWith('/api/')) {
        event.respondWith(networkFirst(request));
        return;
    }

    // Cache-first strategy for static assets
    event.respondWith(cacheFirst(request));
});

/**
 * Cache-first strategy: Check cache, fallback to network
 */
async function cacheFirst(request) {
    const cache = await caches.open(STATIC_CACHE);
    const cached = await cache.match(request);

    if (cached) {
        console.log('[ServiceWorker] Serving from cache:', request.url);
        return cached;
    }

    try {
        console.log('[ServiceWorker] Fetching from network:', request.url);
        const response = await fetch(request);

        // Cache successful responses
        if (response && response.status === 200) {
            const responseToCache = response.clone();
            const dynamicCache = await caches.open(DYNAMIC_CACHE);
            await dynamicCache.put(request, responseToCache);
        }

        return response;
    } catch (error) {
        console.error('[ServiceWorker] Fetch failed:', error);

        // Try to serve from dynamic cache as last resort
        const dynamicCache = await caches.open(DYNAMIC_CACHE);
        const cachedFallback = await dynamicCache.match(request);

        if (cachedFallback) {
            return cachedFallback;
        }

        // Return offline page if available
        return new Response('Offline - content not available', {
            status: 503,
            statusText: 'Service Unavailable',
            headers: new Headers({
                'Content-Type': 'text/plain'
            })
        });
    }
}

/**
 * Network-first strategy: Try network, fallback to cache
 */
async function networkFirst(request) {
    try {
        const response = await fetch(request);

        // Cache successful responses
        if (response && response.status === 200) {
            const responseToCache = response.clone();
            const cache = await caches.open(DYNAMIC_CACHE);
            await cache.put(request, responseToCache);
        }

        return response;
    } catch (error) {
        console.log('[ServiceWorker] Network failed, trying cache:', request.url);

        const cache = await caches.open(DYNAMIC_CACHE);
        const cached = await cache.match(request);

        if (cached) {
            return cached;
        }

        // Return error response
        return new Response(JSON.stringify({
            error: 'Offline',
            message: 'This request requires an internet connection'
        }), {
            status: 503,
            statusText: 'Service Unavailable',
            headers: new Headers({
                'Content-Type': 'application/json'
            })
        });
    }
}

// Background sync for queued operations
self.addEventListener('sync', (event) => {
    console.log('[ServiceWorker] Background sync triggered:', event.tag);

    if (event.tag === 'sync-pod-operations') {
        event.waitUntil(syncPodOperations());
    }
});

/**
 * Sync queued Pod operations when back online
 */
async function syncPodOperations() {
    console.log('[ServiceWorker] Syncing queued Pod operations...');

    // Send message to clients to trigger sync
    const clients = await self.clients.matchAll();
    clients.forEach(client => {
        client.postMessage({
            type: 'SYNC_TRIGGERED',
            timestamp: Date.now()
        });
    });
}

// Listen for messages from clients
self.addEventListener('message', (event) => {
    console.log('[ServiceWorker] Message received:', event.data);

    if (event.data && event.data.type === 'SKIP_WAITING') {
        self.skipWaiting();
    }

    if (event.data && event.data.type === 'CLEAR_CACHE') {
        event.waitUntil(
            caches.keys().then((cacheNames) => {
                return Promise.all(
                    cacheNames.map((cacheName) => {
                        if (cacheName.startsWith('solid-poc-')) {
                            return caches.delete(cacheName);
                        }
                    })
                );
            })
        );
    }
});

console.log('[ServiceWorker] Script loaded');
