// Entry point for bundling Solid client libraries

import * as solidClientAuthentication from '@inrupt/solid-client-authn-browser';
import * as solidClient from '@inrupt/solid-client';

// Export to window object for use in HTML
window.solidClientAuthentication = solidClientAuthentication;
window.solidClient = solidClient;

console.log('Solid client libraries loaded successfully');
