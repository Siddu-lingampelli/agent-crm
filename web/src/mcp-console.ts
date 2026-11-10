export const MCP_ENDPOINT = '/api/v1/mcp';
export const MCP_PROTOCOL = '2025-03-26';

export type JsonObject = Record<string, unknown>;
export interface ToolSchema extends JsonObject {
  type?: string;
  properties?: Record<string, JsonObject>;
  required?: string[];
}
export interface ConsoleTool {
  name: string;
  description?: string;
  inputSchema?: ToolSchema;
  annotations?: { readOnlyHint?: boolean; destructiveHint?: boolean };
}
export interface RpcRequest {
  jsonrpc: '2.0';
  id?: string | number;
  method: string;
  params?: JsonObject;
}
export interface RpcResponse extends JsonObject {
  jsonrpc: '2.0';
  id?: string | number | null;
  result?: JsonObject;
  error?: { code?: number; message: string; data?: unknown };
}
export interface ConsoleRequest {
  id: number;
  request: RpcRequest;
  response: RpcResponse | null;
  startedAt: string;
  latencyMs?: number;
  status: 'pending' | 'success' | 'error';
  error?: string;
}

export class McpRequestError extends Error {
  constructor(message: string, public response: RpcResponse | null = null, public status?: number) {
    super(message); this.name = 'McpRequestError';
  }
}
export function isObject(value: unknown): value is JsonObject {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}
export function parseArguments(source: string, tool?: ConsoleTool | null): JsonObject {
  let parsed: unknown;
  try { parsed = JSON.parse(source); } catch { throw new Error('Enter valid JSON. Check for missing quotes, commas, or brackets.'); }
  if (!isObject(parsed)) throw new Error('Arguments must be a JSON object, such as {}.');
  for (const key of tool?.inputSchema?.required ?? []) {
    if (!(key in parsed)) throw new Error(`Add the required argument "${key}".`);
  }
  for (const [key, definition] of Object.entries(tool?.inputSchema?.properties ?? {})) {
    if (!(key in parsed)) continue;
    const value = parsed[key];
    const type = definition.type;
    if (typeof type === 'string' && !(
      type === 'object' ? isObject(value) : type === 'array' ? Array.isArray(value) :
      type === 'integer' ? typeof value === 'number' && Number.isInteger(value) : type === 'null' ? value === null : typeof value === type
    )) throw new Error(`"${key}" must be ${type === 'integer' ? 'an integer' : `a ${type}`}.`);
    if (Array.isArray(definition.enum) && !definition.enum.includes(value)) throw new Error(`Choose an allowed value for "${key}".`);
    if (typeof value === 'string' && typeof definition.minLength === 'number' && value.length < definition.minLength) throw new Error(`"${key}" needs at least ${definition.minLength} characters.`);
    if (typeof value === 'number' && typeof definition.minimum === 'number' && value < definition.minimum) throw new Error(`"${key}" must be at least ${definition.minimum}.`);
    if (typeof value === 'number' && typeof definition.maximum === 'number' && value > definition.maximum) throw new Error(`"${key}" must be no more than ${definition.maximum}.`);
  }
  return parsed;
}
export function parseRawRequest(source: string): RpcRequest {
  const value = parseArguments(source);
  if (value.jsonrpc !== '2.0' || typeof value.method !== 'string' || !value.method.trim()) throw new Error('Use a JSON-RPC 2.0 object with a method name.');
  if ('id' in value && typeof value.id !== 'number' && typeof value.id !== 'string') throw new Error('Request id must be a string or number. Omit it to send a notification.');
  if ('params' in value && !isObject(value.params)) throw new Error('Request params must be a JSON object.');
  return value as unknown as RpcRequest;
}
export function isReadTool(tool: ConsoleTool) {
  if (typeof tool.annotations?.readOnlyHint === 'boolean') return tool.annotations.readOnlyHint;
  return /(^|\.)(list|get|search|summary|metrics|status|health)$/.test(tool.name);
}
export function exampleArguments(tool: ConsoleTool) {
  const result: JsonObject = {};
  const properties = tool.inputSchema?.properties ?? {};
  for (const key of tool.inputSchema?.required ?? []) {
    const property = properties[key] ?? {};
    result[key] = property.default ?? (Array.isArray(property.enum) ? property.enum[0] : property.type === 'number' || property.type === 'integer' ? 0 : property.type === 'boolean' ? false : property.type === 'array' ? [] : property.type === 'object' ? {} : '');
  }
  if (properties.limit && !('limit' in result)) result.limit = 10;
  return JSON.stringify(result, null, 2);
}
export function resultOutput(response: RpcResponse | null): string {
  if (!response) return '';
  if (response.error) return JSON.stringify(response.error, null, 2);
  const result = response.result;
  if (!result) return JSON.stringify(response, null, 2);
  if (result.structuredContent !== undefined) return JSON.stringify(result.structuredContent, null, 2);
  if (Array.isArray(result.content)) return result.content.map(part => {
    if (isObject(part) && typeof part.text === 'string') {
      try { return JSON.stringify(JSON.parse(part.text), null, 2); } catch { return part.text; }
    }
    return JSON.stringify(part, null, 2);
  }).join('\n\n');
  return JSON.stringify(result, null, 2);
}

/** The local server uses stateless Streamable HTTP with JSON responses. */
export async function sendMcp(request: RpcRequest, options: { protocol?: string; signal?: AbortSignal } = {}): Promise<RpcResponse | null> {
  let response: Response;
  try {
    response = await fetch(MCP_ENDPOINT, {
      method: 'POST', credentials: 'include',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json, text/event-stream', 'MCP-Protocol-Version': options.protocol || MCP_PROTOCOL },
      body: JSON.stringify(request), signal: options.signal ?? AbortSignal.timeout(30000),
    });
  } catch (error) {
    if (error instanceof Error && (error.name === 'TimeoutError' || error.name === 'AbortError')) throw new McpRequestError('The request did not finish. Check the record before retrying a write action.');
    throw new McpRequestError('Could not reach the MCP server. Check your connection and try again.');
  }
  if (response.ok && !('id' in request) && (response.status === 202 || response.status === 204)) return null;
  let data: unknown;
  const body = await response.text();
  try { data = JSON.parse(body); } catch {
    const detail = !response.ok && !/[<>]/.test(body) ? body.trim().slice(0, 500) : '';
    throw new McpRequestError(detail ? `HTTP ${response.status}: ${detail}` : `The server returned an unexpected response (HTTP ${response.status}).`, null, response.status);
  }
  if (!response.ok) {
    const message = response.status === 401 ? 'Your session has expired. Sign in again to reconnect.' : response.status === 403 ? 'Your account does not have access to this action.' : isObject(data) && isObject(data.error) && typeof data.error.message === 'string' ? data.error.message : `Request failed (HTTP ${response.status}).`;
    throw new McpRequestError(message, isObject(data) ? data as RpcResponse : null, response.status);
  }
  if (!isObject(data) || data.jsonrpc !== '2.0') throw new McpRequestError('The server response was not a JSON-RPC 2.0 message.');
  const envelope = data as RpcResponse;
  if ('id' in request && envelope.id !== request.id) throw new McpRequestError('The server response did not match this request.', envelope);
  if (envelope.error) throw new McpRequestError(envelope.error.message, envelope);
  if (envelope.result?.isError === true) throw new McpRequestError(resultOutput(envelope) || 'The tool reported an error.', envelope);
  return envelope;
}
