/**
 * Code Index Client - REST API ONLY
 *
 * This file re-exports restCodeClient to maintain backward compatibility.
 * ALL code index operations now use REST API instead of direct MCP calls.
 *
 * Architecture: UI → REST API → Storage Layer (NO MCP proxying)
 */

export { restCodeClient as codeClient } from './restCodeClient';
